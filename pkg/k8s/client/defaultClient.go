package client

import (
	"walm/pkg/setting"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	. "walm/pkg/util/log"
	clientsetex "transwarp/application-instance/pkg/client/clientset/versioned"
	"k8s.io/helm/pkg/kube"
)

var DefaultApiserverClient *kubernetes.Clientset
var DefaultRestConfig *restclient.Config
var DefaultApiserverClientEx *clientsetex.Clientset
var DefaultKubeClient *kube.Client

func GetDefaultClient() *kubernetes.Clientset {
	var err error
	if DefaultApiserverClient == nil {
		DefaultApiserverClient, err = createApiserverClient(setting.Config.Kube.MasterHost, setting.Config.Kube.Config)
	}
	if err != nil {
		Log.Fatalf("create apiserver client failed:%v", err)
	}
	return DefaultApiserverClient
}

func GetDefaultClientEx() *clientsetex.Clientset {
	if DefaultApiserverClientEx == nil {
		var err error
		DefaultApiserverClientEx, err = createApiserverClientEx(setting.Config.Kube.MasterHost, setting.Config.Kube.Config)
		if err != nil {
			Log.Fatalf("create apiserver client failed:%v", err)
		}
	}

	return DefaultApiserverClientEx
}

func GetDefaultRestConfig() *restclient.Config {
	var err error
	if DefaultRestConfig == nil {
		DefaultRestConfig, err = clientcmd.BuildConfigFromFlags(setting.Config.Kube.MasterHost, setting.Config.Kube.Config)
	}
	if err != nil {
		Log.Fatalf("get default rest config= failed:%v", err)
	}
	return DefaultRestConfig
}


func GetKubeClient() *kube.Client {

	if DefaultKubeClient == nil {
		DefaultKubeClient = CreateKubeClient(setting.Config.Kube.MasterHost,  setting.Config.Kube.Config)
	}

	return  DefaultKubeClient
}

