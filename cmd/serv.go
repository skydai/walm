package cmd

import (
	migrationhttp "WarpCloud/walm/pkg/crd/migration/delivery/http"
	helmImpl "WarpCloud/walm/pkg/helm/impl"
	cacheInformer "WarpCloud/walm/pkg/k8s/cache/informer"
	"WarpCloud/walm/pkg/k8s/client"
	k8sHelm "WarpCloud/walm/pkg/k8s/client/helm"
	"WarpCloud/walm/pkg/k8s/elect"
	"WarpCloud/walm/pkg/k8s/operator"
	kafkaimpl "WarpCloud/walm/pkg/kafka/impl"
	httpModel "WarpCloud/walm/pkg/models/http"
	nodehttp "WarpCloud/walm/pkg/node/delivery/http"
	podhttp "WarpCloud/walm/pkg/pod/delivery/http"
	projectcache "WarpCloud/walm/pkg/project/cache/redis"
	projecthttp "WarpCloud/walm/pkg/project/delivery/http"
	projectusecase "WarpCloud/walm/pkg/project/usecase"
	pvchttp "WarpCloud/walm/pkg/pvc/delivery/http"
	"WarpCloud/walm/pkg/redis/impl"
	releasecache "WarpCloud/walm/pkg/release/cache/redis"
	releaseconfig "WarpCloud/walm/pkg/release/config"
	releasehttp "WarpCloud/walm/pkg/release/delivery/http"
	releaseusecase "WarpCloud/walm/pkg/release/usecase/helm"
	secrethttp "WarpCloud/walm/pkg/secret/delivery/http"
	"WarpCloud/walm/pkg/setting"
	storageclasshttp "WarpCloud/walm/pkg/storageclass/delivery/http"
	"WarpCloud/walm/pkg/sync"
	"WarpCloud/walm/pkg/task/machinery"
	tenanthttp "WarpCloud/walm/pkg/tenant/delivery/http"
	tenantusecase "WarpCloud/walm/pkg/tenant/usecase"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	migrationclientset "github.com/migration/pkg/client/clientset/versioned"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/thoas/stats"
	"k8s.io/klog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
	instanceclientset "transwarp/application-instance/pkg/client/clientset/versioned"
)

const servDesc = `
This command enable a WALM Web server.

$ walm serv 

Before to start serv ,you need to config the conf file 

The file is named conf.yaml

`

const DefaultElectionId = "walm-election-id"

type ServCmd struct {
	cfgFile string
}

var (
	HTTPReqDuration *prometheus.HistogramVec
	HTTPReqTotal    *prometheus.CounterVec
)

func NewServCmd() *cobra.Command {
	inst := &ServCmd{}

	cmd := &cobra.Command{
		Use:   "serv",
		Short: "enable a Walm Web Server",
		Long:  servDesc,

		RunE: func(cmd *cobra.Command, args []string) error {
			return inst.run()
		},
	}
	cmd.PersistentFlags().StringVar(&inst.cfgFile, "config", "walm.yaml", "config file (default is walm.yaml)")

	return cmd
}

func (sc *ServCmd) run() error {

	sig := make(chan os.Signal, 1)

	sc.initConfig()
	if os.Getenv("Pod_Name") != "" {
		setting.Config.ElectorConfig.LockIdentity = os.Getenv("Pod_Name")
	}
	if os.Getenv("Pod_Namespace") != "" {
		setting.Config.ElectorConfig.LockNamespace = os.Getenv("Pod_Namespace")
	}
	if setting.Config.ElectorConfig.LockNamespace == "" || setting.Config.ElectorConfig.LockIdentity == "" {
		err := errors.New("both env var lockNamespace and lockIdentity must not be empty")
		klog.Error(err.Error())
		return err
	}
	if setting.Config.ElectorConfig.ElectionId == "" {
		setting.Config.ElectorConfig.ElectionId = DefaultElectionId
	}
	config := setting.Config
	initLogLevel()
	stopChan := make(chan struct{})

	kubeConfig := ""
	if config.KubeConfig != nil {
		kubeConfig = config.KubeConfig.Config
	}
	kubeContest := ""
	if config.KubeConfig != nil {
		kubeContest = config.KubeConfig.Context
	}
	k8sClient, err := client.NewClient("", kubeConfig)
	if err != nil {
		klog.Errorf("failed to create k8s client : %s", err.Error())
		return err
	}
	k8sReleaseConfigClient, err := client.NewReleaseConfigClient("", kubeConfig)
	if err != nil {
		klog.Errorf("failed to create k8s release config client : %s", err.Error())
		return err
	}
	var k8sInstanceClient *instanceclientset.Clientset
	if config.CrdConfig == nil || !config.CrdConfig.NotNeedInstance {
		klog.Info("CRD ApplicationInstance should be installed in the k8s")
		k8sInstanceClient, err = client.NewInstanceClient("", kubeConfig)
		if err != nil {
			klog.Errorf("failed to create k8s instance client : %s", err.Error())
			return err
		}
	}
	var k8sMigrationClient *migrationclientset.Clientset
	if config.CrdConfig != nil && config.CrdConfig.EnableMigrationCRD {
		klog.Info("CRD ApplicationInstance should be installed in the k8s")
		k8sMigrationClient, err = client.NewMigrationClient("", kubeConfig)
		if err != nil {
			klog.Errorf("failed to create k8s instance client : %s", err.Error())
			return err
		}
	}

	k8sCache := cacheInformer.NewInformer(k8sClient, k8sReleaseConfigClient, k8sInstanceClient, k8sMigrationClient, 0, stopChan)

	if config.TaskConfig == nil {
		err = errors.New("task config can not be empty")
		klog.Error(err.Error())
		return err
	}
	task, err := machinery.NewTask(config.TaskConfig)
	if err != nil {
		klog.Errorf("failed to create task manager %s", err.Error())
		return err
	}

	registryClient, err := helmImpl.NewRegistryClient(config.ChartImageConfig)
	if err != nil {
		klog.Errorf("failed to create registry client : %s", err.Error())
		return err
	}
	kubeClients := k8sHelm.NewHelmKubeClient(kubeConfig, kubeContest, k8sInstanceClient)
	helm, err := helmImpl.NewHelm(config.RepoList, registryClient, k8sCache, kubeClients)
	if err != nil {
		klog.Errorf("failed to create helm manager: %s", err.Error())
		return err
	}
	k8sOperator := operator.NewOperator(k8sClient, k8sCache, kubeClients, k8sMigrationClient)
	if config.RedisConfig == nil {
		err = errors.New("redis config can not be empty")
		klog.Error(err.Error())
		return err
	}
	redisClient := impl.NewRedisClient(config.RedisConfig)
	redis := impl.NewRedis(redisClient)
	redisEx := impl.NewRedisEx(config.RedisConfig, time.Second * 30)
	releaseCache := releasecache.NewCache(redis)
	releaseUseCase, err := releaseusecase.NewHelm(releaseCache, helm, k8sCache, k8sOperator, task, redisEx)
	if err != nil {
		klog.Errorf("failed to new release use case : %s", err.Error())
		return err
	}
	projectCache := projectcache.NewProjectCache(redis)
	projectUseCase, err := projectusecase.NewProject(projectCache, task, releaseUseCase, helm)
	if err != nil {
		klog.Errorf("failed to new project use case : %s", err.Error())
		return err
	}

	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		select {
		case <-stopChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	syncManager := sync.NewSync(redisClient, helm, k8sCache, task, "", "", "")
	kafka, err := kafkaimpl.NewKafka(config.KafkaConfig)
	if err != nil {
		klog.Errorf("failed to create kafka manager: %s", err.Error())
		return err
	}
	releaseConfigController := releaseconfig.NewReleaseConfigController(k8sCache, releaseUseCase, kafka, 0)
	onStartedLeadingFunc := func(context context.Context) {
		klog.Info("Succeed to elect leader")
		syncManager.Start(context.Done())
		releaseConfigController.Start(context.Done())
	}
	onNewLeaderFunc := func(identity string) {
		klog.Infof("Now leader is changed to %s", identity)
	}
	onStoppedLeadingFunc := func() {
		klog.Info("Stopped being a leader")
		sig <- syscall.SIGINT
	}

	electorConfig := &elect.ElectorConfig{
		LockNamespace:        config.ElectorConfig.LockNamespace,
		LockIdentity:         config.ElectorConfig.LockIdentity,
		ElectionId:           config.ElectorConfig.ElectionId,
		Client:               k8sClient,
		OnStartedLeadingFunc: onStartedLeadingFunc,
		OnNewLeaderFunc:      onNewLeaderFunc,
		OnStoppedLeadingFunc: onStoppedLeadingFunc,
	}

	elector, err := elect.NewElector(electorConfig)
	if err != nil {
		klog.Errorf("create leader elector failed")
		return err
	}
	klog.Info("Start to elect leader")
	go elector.Run(ctx)

	HTTPReqDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "http_request_duration_seconds",
		Help:        "The HTTP request latencies represent with seconds.",
		Buckets:     nil,
	}, []string{"method", "path"})

	HTTPReqTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests made.",
	}, []string{"method", "path", "status"})

	prometheus.MustRegister(
		HTTPReqTotal,
		HTTPReqDuration,
	)
	restful.DefaultRequestContentType(restful.MIME_JSON)
	restful.DefaultResponseContentType(restful.MIME_JSON)
	// gzip if accepted
	restful.DefaultContainer.EnableContentEncoding(true)
	// faster router
	restful.DefaultContainer.Router(restful.CurlyRouter{})
	restful.Filter(ServerStatsFilter)
	restful.Filter(RouteLogging)
	klog.Infoln("Adding Route...")

	restful.Add(InitRootRouter())
	restful.Add(nodehttp.RegisterNodeHandler(k8sCache, k8sOperator))
	restful.Add(migrationhttp.RegisterCrdHandler(k8sCache, k8sOperator))
	restful.Add(secrethttp.RegisterSecretHandler(k8sCache, k8sOperator))
	restful.Add(storageclasshttp.RegisterStorageClassHandler(k8sCache))
	restful.Add(pvchttp.RegisterPvcHandler(k8sCache, k8sOperator))
	tenantUseCase := tenantusecase.NewTenant(k8sCache, k8sOperator, releaseUseCase)
	restful.Add(tenanthttp.RegisterTenantHandler(tenantUseCase))
	restful.Add(projecthttp.RegisterProjectHandler(projecthttp.NewProjectHandler(projectUseCase)))
	restful.Add(releasehttp.RegisterReleaseHandler(releasehttp.NewReleaseHandler(releaseUseCase)))
	restful.Add(podhttp.RegisterPodHandler(k8sCache, k8sOperator))
	restful.Add(releasehttp.RegisterChartHandler(helm))
	klog.Infoln("Add Route Success")
	restConfig := restfulspec.Config{
		// You control what services are visible
		WebServices:                   restful.RegisteredWebServices(),
		APIPath:                       "/apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject}
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(restConfig))
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir("swagger-ui/dist"))))
	http.Handle("/swagger/", http.RedirectHandler("/swagger-ui/?url=/apidocs.json", http.StatusFound))
	klog.Infof("ready to serve on port %d", setting.Config.HttpConfig.HTTPPort)

	if setting.Config.Debug {
		go func() {
			klog.Info("supporting pprof on port 6060...")
			klog.Error(http.ListenAndServe(":6060", nil))
		}()
	}

	server := &http.Server{Addr: fmt.Sprintf(":%d", setting.Config.HttpConfig.HTTPPort), Handler: restful.DefaultContainer}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			klog.Error(err.Error())
			sig <- syscall.SIGINT
		}
	}()

	// make sure worker starts after all tasks registered
	task.StartWorker()

	//shut down gracefully
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	err = server.Shutdown(context.Background())
	if err != nil {
		klog.Error(err.Error())
	}
	close(stopChan)
	task.StopWorker(30)
	klog.Info("waiting for informer stopping")
	time.Sleep(2 * time.Second)
	klog.Info("walm server stopped gracefully")
	return nil
}

func RouteLogging(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	now := time.Now()
	chain.ProcessFilter(req, resp)

	duration := time.Now().Sub(now)
	HTTPReqTotal.With(prometheus.Labels{
		"method": req.Request.Method,
		"path":   req.Request.URL.Path,
		"status": string(resp.StatusCode()),
	}).Inc()
//	float64(time.Since(tBegin)) / float64(time.Millisecond)
	HTTPReqDuration.With(prometheus.Labels{
		"method": req.Request.Method,
		"path":   req.Request.URL.Path,
	}).Observe(float64(duration)/float64(time.Second))
	klog.Infof("[route-filter (logger)] CLIENT %s OP %s URI %s COST %v RESP %d", req.Request.Host, req.Request.Method, req.Request.URL, duration, resp.StatusCode())
}

var ServerStats = stats.New()

func ServerStatsFilter(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
	beginning, recorder := ServerStats.Begin(response)
	chain.ProcessFilter(request, response)
	ServerStats.End(beginning, stats.WithRecorder(recorder))
}

func ServerStatsData(request *restful.Request, response *restful.Response) {
	response.WriteEntity(ServerStats.Data())
}

func readinessProbe(request *restful.Request, response *restful.Response) {
	response.WriteEntity("OK")
}

func livenessProbe(request *restful.Request, response *restful.Response) {
	response.WriteEntity("OK")
}

func InitRootRouter() *restful.WebService {
	ws := new(restful.WebService)

	ws.Path("/").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	tags := []string{"root"}

	ws.Route(ws.GET("/readiness").To(readinessProbe).
		Doc("服务Ready状态检查").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", httpModel.ErrorMessageResponse{}))

	ws.Route(ws.GET("/liveniess").To(livenessProbe).
		Doc("服务Live状态检查").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", httpModel.ErrorMessageResponse{}))

	ws.Route(ws.GET("/stats").To(ServerStatsData).
		Doc("获取服务Stats").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Returns(200, "OK", nil).
		Returns(500, "Internal Error", httpModel.ErrorMessageResponse{}))

	return ws
}

func initLogLevel() {
	if setting.Config.LogConfig != nil {
		if setting.Config.LogConfig.Level == "debug" {
			pflag.CommandLine.Set("v", "2")
		}
	}
}

func (sc *ServCmd) initConfig() {
	klog.Infof("loading configuration from [%s]", sc.cfgFile)
	setting.InitConfig(sc.cfgFile)
	settingConfig, err := json.MarshalIndent(setting.Config, "", "  ")
	if err != nil {
		klog.Fatal("failed to marshal setting config")
	}
	klog.Infof("finished loading configuration:\n%s", string(settingConfig))
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "Walm",
			Description: "Walm Web Server",
			Version:     "0.0.1",
		},
	}
}
