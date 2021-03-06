package operator

import (
	"WarpCloud/walm/pkg/k8s"
	"WarpCloud/walm/pkg/k8s/client/helm"
	"WarpCloud/walm/pkg/k8s/converter"
	"WarpCloud/walm/pkg/k8s/utils"
	errorModel "WarpCloud/walm/pkg/models/error"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	tosv1beta1 "github.com/migration/pkg/apis/tos/v1beta1"
	migrationclientset "github.com/migration/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"reflect"
	"strconv"
	"strings"
	releaseconfigclientset "transwarp/release-config/pkg/client/clientset/versioned"
)

const (
	storageClassAnnotationKey = "volume.beta.kubernetes.io/storage-class"
)

type Operator struct {
	client             *kubernetes.Clientset
	k8sCache           k8s.Cache
	kubeClients        *helm.Client
	k8sMigrationClient *migrationclientset.Clientset
	k8sReleaseConfigClient *releaseconfigclientset.Clientset
}

func (op *Operator) DeleteStatefulSetPvcs(statefulSets []*k8sModel.StatefulSet) error {
	for _, statefulSet := range statefulSets {
		pvcs, err := op.k8sCache.ListPersistentVolumeClaims(statefulSet.Namespace, statefulSet.Selector)
		if err != nil {
			klog.Errorf("failed to list pvcs : %s", err.Error())
			return err
		}
		for _, pvc := range pvcs {
			klog.Infof("start to delete statefulSet pvc %s/%s", pvc.Namespace, pvc.Name)
			err := op.doDeletePvc(pvc, true)
			if err != nil {
				return err
			}
			klog.Infof("succeed to delete statefulSet pvc %s/%s", pvc.Namespace, pvc.Name)
		}
	}
	return nil
}

func (op *Operator) DeleteIsomateSetPvcs(isomateSets []*k8sModel.IsomateSet) error {
	for _, isomateSet := range isomateSets {
		pvcs, err := op.k8sCache.ListPersistentVolumeClaims(isomateSet.Namespace, isomateSet.Selector)
		if err != nil {
			klog.Errorf("failed to list pvcs : %s", err.Error())
			return err
		}
		for _, pvc := range pvcs {
			klog.Infof("start to delete isomateSet pvc %s/%s", pvc.Namespace, pvc.Name)
			err := op.doDeletePvc(pvc, true)
			if err != nil {
				return err
			}
			klog.Infof("succeed to delete isomateSet pvc %s/%s", pvc.Namespace, pvc.Name)
		}
	}
	return nil
}

func (op *Operator) DeletePod(namespace string, name string) error {
	err := op.client.CoreV1().Pods(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("pod %s/%s is not found ", namespace, name)
			return nil
		}
		klog.Errorf("failed to delete pod %s/%s : %s", namespace, name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) RestartPod(namespace string, name string) error {
	err := op.client.CoreV1().Pods(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("failed to restart pod %s/%s : %s", namespace, name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) MigratePod(mig *k8sModel.Mig) error {

	err := op.DeletePodMigration(mig.Spec.Namespace, mig.Spec.PodName)
	if err != nil {
		return err
	}
	k8sMig, err := converter.ConvertMigToK8s(mig)
	if err != nil {
		return errors.Errorf("failed to convert mig to k8sMigration: %s", err.Error())
	}

	_, err = op.k8sMigrationClient.ApiextensionsV1beta1().Migs(mig.Namespace).Create(k8sMig)
	if err != nil {
		return errors.Errorf("failed to migrate pod: %s", err.Error())
	}
	return nil
}

func (op *Operator) DeletePodMigration(namespace string, name string) error {

	migName := "mig" + "-" + namespace + "-" + name
	err := op.k8sMigrationClient.ApiextensionsV1beta1().Migs("default").Delete(migName, &metav1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		} else {
			klog.Errorf("failed to delete pod migration: %s", err.Error())
			return err
		}
	}
	return nil
}

func (op *Operator) MigrateNode(srcNode string, destNode string) error {

	/* node check */
	k8sMigrationClient := op.k8sMigrationClient
	if k8sMigrationClient == nil {
		return errors.Errorf("failed to get migration client, check config.CrdConfig.EnableMigrationCRD")
	}

	if destNode != "" {
		dest, err := op.k8sCache.GetResource(k8sModel.NodeKind, "", destNode)
		if err != nil {
			klog.Errorf("failed to get node %s: %s", dest, err.Error())
			return err
		}
		newDest := dest.(*k8sModel.Node)
		if newDest.UnSchedulable {
			return errors.Errorf("dest node is unschedulable, please check")
		}
	}

	/* cordon node */
	src, err := op.client.CoreV1().Nodes().Get(srcNode, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get node %s: %s", srcNode, err.Error())
		return err
	}
	if src.Spec.Unschedulable == false {
		oldData, err := json.Marshal(srcNode)
		if err != nil {
			return err
		}

		src.Spec.Unschedulable = true
		newData, err := json.Marshal(srcNode)
		if err != nil {
			return err
		}
		patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, srcNode)
		if patchErr == nil {
			_, err = op.client.CoreV1().Nodes().Patch(src.Name, types.StrategicMergePatchType, patchBytes)

		} else {
			_, err = op.client.CoreV1().Nodes().Update(src)
		}
		if err != nil {
			klog.Errorf("error: unable to cordon node %q: %v\n", src.Name, err)
			return err
		}
	} else {
		klog.Infof("node %s is unschedulable now", src.Name)
	}

	/*  get pods to be migrated && pre-check */
	var podList []*k8sModel.Pod
	statefulsets, err := op.k8sCache.ListStatefulSets("", "")
	if err != nil {
		klog.Errorf("failed to get sts: %s", err.Error())
		return err
	}
	for _, sts := range statefulsets {
		for _, pod := range sts.Pods {
			if pod.NodeName == srcNode {
				if err = op.migratePodPreCheck(pod.Namespace, pod.Name); err != nil {
					return err
				}
				podList = append(podList, pod)
			}
		}
	}

	for _, pod := range podList {
		mig := &k8sModel.Mig{
			Meta: k8sModel.Meta{
				Namespace: "default",
				Name:      "mig" + "-" + pod.Namespace + "-" + pod.Name,
			},
			Labels: map[string]string{"migType": "node", "srcNode": srcNode},
			Spec: k8sModel.MigSpec{
				Namespace: pod.Namespace,
				PodName:   pod.Name,
			},
			SrcHost:  srcNode,
			DestHost: destNode,
		}
		err = op.MigratePod(mig)
		if err != nil {
			return err
		}
	}

	return nil
}

func (op *Operator) migratePodPreCheck(namespace string, name string) error {
	resource, err := op.k8sCache.GetResource(k8sModel.MigKind, namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	mig := resource.(*k8sModel.Mig)
	switch mig.State.Status {
	case tosv1beta1.MIG_CREATED, tosv1beta1.MIG_IN_PROGRESS, "":
		return errors.Errorf("Pod %s/%s migration in progress, please wait for the last process end.", namespace, name)
	case tosv1beta1.MIG_FINISH:
	case tosv1beta1.MIG_FAILED:
		err = op.DeletePodMigration(namespace, name)
		if err != nil {
			return err
		}
		return errors.Errorf("Last migration for pod %s/%s failed: %s\nThe failed mig has been deleted, fix error and retry", namespace, name, mig.State.Message)
	}
	return nil
}

func (op *Operator) BuildManifestObjects(namespace string, manifest string) ([]map[string]interface{}, error) {
	_, kubeClient := op.kubeClients.GetKubeClient(namespace)
	resources, err := kubeClient.Build(bytes.NewBufferString(manifest))
	if err != nil {
		klog.Errorf("failed to build unstructured : %s", err.Error())
		return nil, err
	}

	results := []map[string]interface{}{}
	for _, resource := range resources {
		results = append(results, resource.Object.(*unstructured.Unstructured).Object)
	}
	return results, nil
}

func (op *Operator) ComputeReleaseResourcesByManifest(namespace string, manifest string) (*release.ReleaseResources, error) {
	_, kubeClient := op.kubeClients.GetKubeClient(namespace)
	resources, err := kubeClient.Build(bytes.NewBufferString(manifest))
	if err != nil {
		klog.Errorf("failed to build unstructured : %s", err.Error())
		return nil, err
	}

	result := &release.ReleaseResources{}
	for _, resource := range resources {
		unstructured := resource.Object.(*unstructured.Unstructured)
		switch unstructured.GetKind() {
		case "Deployment":
			releaseResourceDeployment, err := buildReleaseResourceDeployment(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource deployment %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.Deployments = append(result.Deployments, releaseResourceDeployment)
		case "StatefulSet":
			releaseResourceStatefulSet, err := buildReleaseResourceStatefulSet(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource stateful set %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.StatefulSets = append(result.StatefulSets, releaseResourceStatefulSet)
		case "DaemonSet":
			releaseResourceDaemonSet, err := buildReleaseResourceDaemonSet(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource daemon set %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.DaemonSets = append(result.DaemonSets, releaseResourceDaemonSet)
		case "Job":
			releaseResourceJob, err := buildReleaseResourceJob(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource job %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.Jobs = append(result.Jobs, releaseResourceJob)
		case "PersistentVolumeClaim":
			pvc, err := buildReleaseResourcePvc(unstructured)
			if err != nil {
				klog.Errorf("failed to build release resource pvc %s : %s", unstructured.GetName(), err.Error())
				return nil, err
			}
			result.Pvcs = append(result.Pvcs, pvc)
		default:
		}
	}
	return result, nil
}

func buildReleaseResourceDeployment(resource *unstructured.Unstructured) (*release.ReleaseResourceDeployment, error) {
	deployment := &v1beta1.Deployment{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal deployment %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, deployment)
	if err != nil {
		klog.Errorf("failed to unmarshal deployment %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResourceDeployment := &release.ReleaseResourceDeployment{}
	if deployment.Spec.Replicas != nil {
		releaseResourceDeployment.Replicas = *deployment.Spec.Replicas
	}

	releaseResourceDeployment.ReleaseResourceBase, err = buildReleaseResourceBase(resource, deployment.Spec.Template, nil)
	if err != nil {
		klog.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResourceDeployment, nil
}

func buildReleaseResourceStatefulSet(resource *unstructured.Unstructured) (*release.ReleaseResourceStatefulSet, error) {
	statefulSet := &appsv1beta1.StatefulSet{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal statefulSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, statefulSet)
	if err != nil {
		klog.Errorf("failed to unmarshal statefulSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResource := &release.ReleaseResourceStatefulSet{}
	if statefulSet.Spec.Replicas != nil {
		releaseResource.Replicas = *statefulSet.Spec.Replicas
	}

	releaseResource.ReleaseResourceBase, err = buildReleaseResourceBase(resource, statefulSet.Spec.Template, statefulSet.Spec.VolumeClaimTemplates)
	if err != nil {
		klog.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResource, nil
}

func buildReleaseResourceDaemonSet(resource *unstructured.Unstructured) (*release.ReleaseResourceDaemonSet, error) {
	daemonSet := &extv1beta1.DaemonSet{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal daemonSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, daemonSet)
	if err != nil {
		klog.Errorf("failed to unmarshal daemonSet %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResource := &release.ReleaseResourceDaemonSet{
		NodeSelector: daemonSet.Spec.Template.Spec.NodeSelector,
	}

	releaseResource.ReleaseResourceBase, err = buildReleaseResourceBase(resource, daemonSet.Spec.Template, nil)
	if err != nil {
		klog.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResource, nil
}

func buildReleaseResourceJob(resource *unstructured.Unstructured) (*release.ReleaseResourceJob, error) {
	job := &batchv1.Job{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal job %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, job)
	if err != nil {
		klog.Errorf("failed to unmarshal job %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	releaseResource := &release.ReleaseResourceJob{}
	if job.Spec.Parallelism != nil {
		releaseResource.Parallelism = *job.Spec.Parallelism
	}
	if job.Spec.Completions != nil {
		releaseResource.Completions = *job.Spec.Completions
	}

	releaseResource.ReleaseResourceBase, err = buildReleaseResourceBase(resource, job.Spec.Template, nil)
	if err != nil {
		klog.Errorf("failed to build release resource : %s", err.Error())
		return nil, err
	}
	return releaseResource, nil
}

func buildReleaseResourcePvc(resource *unstructured.Unstructured) (*release.ReleaseResourceStorage, error) {
	pvc := &v1.PersistentVolumeClaim{}
	resourceBytes, err := resource.MarshalJSON()
	if err != nil {
		klog.Errorf("failed to marshal pvc %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	err = json.Unmarshal(resourceBytes, pvc)
	if err != nil {
		klog.Errorf("failed to unmarshal pvc %s : %s", resource.GetName(), err.Error())
		return nil, err
	}

	return buildPvcStorage(*pvc), nil
}

func buildReleaseResourceBase(r *unstructured.Unstructured, podTemplateSpec v1.PodTemplateSpec, pvcs []v1.PersistentVolumeClaim) (releaseResource release.ReleaseResourceBase, err error) {
	releaseResource = release.ReleaseResourceBase{
		Name:        r.GetName(),
		PodRequests: &release.ReleaseResourcePod{},
		PodLimits:   &release.ReleaseResourcePod{},
	}

	podRequests, podLimits := utils.GetPodRequestsAndLimits(podTemplateSpec.Spec)
	if quantity, ok := podRequests[v1.ResourceCPU]; ok {
		releaseResource.PodRequests.Cpu = float64(quantity.MilliValue()) / utils.K8sResourceCpuScale
	}
	if quantity, ok := podRequests[v1.ResourceMemory]; ok {
		releaseResource.PodRequests.Memory = quantity.Value() / utils.K8sResourceMemoryScale
	}
	if quantity, ok := podLimits[v1.ResourceCPU]; ok {
		releaseResource.PodLimits.Cpu = float64(quantity.MilliValue()) / utils.K8sResourceCpuScale
	}
	if quantity, ok := podLimits[v1.ResourceMemory]; ok {
		releaseResource.PodLimits.Memory = quantity.Value() / utils.K8sResourceMemoryScale
	}

	releaseResource.PodRequests.Storage = buildTosDiskStorage(r.Object)
	releaseResource.PodRequests.Storage = append(releaseResource.PodRequests.Storage, buildPvcStorages(pvcs)...)
	return
}

func buildTosDiskStorage(object map[string]interface{}) (tosDiskStorages []*release.ReleaseResourceStorage) {
	tosDiskStorages = []*release.ReleaseResourceStorage{}
	type TosDiskVolumeSource struct {
		Name        string        `json:"name" description:"tos disk name"`
		StorageType string        `json:"storageType" description:"tos disk storageType"`
		Capability  v1.Capability `json:"capability" description:"tos disk capability"`
	}

	volumes, found, err := unstructured.NestedSlice(object, "spec", "template", "spec", "volumes")
	if !found || err != nil {
		klog.Warning("failed to find pod volumes")
		return
	}

	for _, volume := range volumes {
		if volumeMap, ok := volume.(map[string]interface{}); ok {
			if tosDisk, ok1 := volumeMap["tosDisk"]; ok1 {
				tosDiskBytes, err := json.Marshal(tosDisk)
				if err != nil {
					klog.Warningf("failed to marshal tosDisk : %s", err.Error())
					continue
				}
				tosDiskVolumeSource := &TosDiskVolumeSource{}
				err = json.Unmarshal(tosDiskBytes, tosDiskVolumeSource)
				if err != nil {
					klog.Warningf("failed to unmarshal tosDisk : %s", err.Error())
					continue
				}

				quantity, err := resource.ParseQuantity(string(tosDiskVolumeSource.Capability))
				if err != nil {
					klog.Warningf("failed to parse quantity: %s", err.Error())
					continue
				}

				tosDiskStorages = append(tosDiskStorages, &release.ReleaseResourceStorage{
					Name:         tosDiskVolumeSource.Name,
					Type:         release.TosDiskPodStorageType,
					Size:         quantity.Value() / utils.K8sResourceStorageScale,
					StorageClass: tosDiskVolumeSource.StorageType,
				})
			}
		}
	}
	return
}

func buildPvcStorages(pvcs []v1.PersistentVolumeClaim) (pvcStorages []*release.ReleaseResourceStorage) {
	pvcStorages = []*release.ReleaseResourceStorage{}
	for _, pvc := range pvcs {
		pvcStorages = append(pvcStorages, buildPvcStorage(pvc))
	}
	return
}

func buildPvcStorage(pvc v1.PersistentVolumeClaim) *release.ReleaseResourceStorage {
	pvcStorage := &release.ReleaseResourceStorage{
		Name: pvc.Name,
		Type: release.PvcPodStorageType,
	}
	quantity := pvc.Spec.Resources.Requests[v1.ResourceStorage]
	pvcStorage.Size = quantity.Value() / utils.K8sResourceStorageScale
	if pvc.Spec.StorageClassName != nil {
		pvcStorage.StorageClass = *pvc.Spec.StorageClassName
	} else if len(pvc.Annotations) > 0 {
		pvcStorage.StorageClass = pvc.Annotations[storageClassAnnotationKey]
	}
	return pvcStorage
}

func (op *Operator) CreateNamespace(namespace *k8sModel.Namespace) error {
	k8sNamespace, err := converter.ConvertNamespaceToK8s(namespace)
	if err != nil {
		klog.Errorf("failed to convert namespace : %s", err.Error())
		return err
	}
	_, err = op.client.CoreV1().Namespaces().Create(k8sNamespace)
	if err != nil {
		klog.Errorf("failed to create namespace %s : %s", k8sNamespace.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) UpdateNamespace(namespace *k8sModel.Namespace) error {
	k8sNamespace, err := converter.ConvertNamespaceToK8s(namespace)
	if err != nil {
		klog.Errorf("failed to convert namespace : %s", err.Error())
		return err
	}
	_, err = op.client.CoreV1().Namespaces().Update(k8sNamespace)
	if err != nil {
		klog.Errorf("failed to update namespace %s : %s", k8sNamespace.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) DeleteNamespace(name string) error {
	err := op.client.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("namespace %s is not found ", name)
			return nil
		}
		klog.Errorf("failed to delete namespace %s : %s", name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) CreateResourceQuota(resourceQuota *k8sModel.ResourceQuota) error {
	k8sQuota, err := converter.ConvertResourceQuotaToK8s(resourceQuota)
	if err != nil {
		klog.Errorf("failed to convert resource quota : %s", err.Error())
		return err
	}
	_, err = op.client.CoreV1().ResourceQuotas(k8sQuota.Namespace).Create(k8sQuota)
	if err != nil {
		klog.Errorf("failed to create resource quota %s/%s : %s", k8sQuota.Namespace, k8sQuota.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) CreateOrUpdateResourceQuota(resourceQuota *k8sModel.ResourceQuota) error {
	update := true
	_, err := op.client.CoreV1().ResourceQuotas(resourceQuota.Namespace).Get(resourceQuota.Name, metav1.GetOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			update = false
		} else {
			klog.Errorf("failed to get resource quota %s/%s : %s", resourceQuota.Namespace, resourceQuota.Name, err.Error())
			return err
		}
	}

	k8sQuota, err := converter.ConvertResourceQuotaToK8s(resourceQuota)
	if err != nil {
		klog.Errorf("failed to convert resource quota : %s", err.Error())
		return err
	}

	if update {
		_, err = op.client.CoreV1().ResourceQuotas(k8sQuota.Namespace).Update(k8sQuota)
		if err != nil {
			klog.Errorf("failed to update resource quota %s/%s : %s", k8sQuota.Namespace, k8sQuota.Name, err.Error())
			return err
		}
	} else {
		_, err = op.client.CoreV1().ResourceQuotas(k8sQuota.Namespace).Create(k8sQuota)
		if err != nil {
			klog.Errorf("failed to create resource quota %s/%s : %s", k8sQuota.Namespace, k8sQuota.Name, err.Error())
			return err
		}
	}
	return nil
}

func (op *Operator) CreateLimitRange(limitRange *k8sModel.LimitRange) error {
	k8sLimitRange, err := converter.ConvertLimitRangeToK8s(limitRange)
	if err != nil {
		klog.Errorf("failed to convert limit range : %s", err.Error())
		return err
	}

	_, err = op.client.CoreV1().LimitRanges(k8sLimitRange.Namespace).Create(k8sLimitRange)
	if err != nil {
		klog.Errorf("failed to create limit range %s/%s : %s", k8sLimitRange.Namespace, k8sLimitRange.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) LabelNode(name string, labelsToAdd map[string]string, labelsToRemove []string) (err error) {
	if len(labelsToAdd) == 0 && len(labelsToRemove) == 0 {
		return
	}

	node, err := op.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	oldLabels, err := json.Marshal(node.Labels)
	if err != nil {
		return
	}

	node.Labels = utils.MergeLabels(node.Labels, labelsToAdd, labelsToRemove)
	newLabels, err := json.Marshal(node.Labels)
	if err != nil {
		return
	}

	if !reflect.DeepEqual(oldLabels, newLabels) {
		_, err = op.client.CoreV1().Nodes().Update(node)
		if err != nil {
			klog.Errorf("failed to update node %s : %s", name, err.Error())
			return
		}
	}

	return
}

func (op *Operator) AnnotateNode(name string, annotationsToAdd map[string]string, annotationsToRemove []string) (err error) {
	if len(annotationsToAdd) == 0 && len(annotationsToRemove) == 0 {
		return
	}

	node, err := op.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	oldAnnos, err := json.Marshal(node.Annotations)
	if err != nil {
		return
	}

	node.Annotations = utils.MergeLabels(node.Annotations, annotationsToAdd, annotationsToRemove)
	newAnnos, err := json.Marshal(node.Annotations)
	if err != nil {
		return
	}

	if !reflect.DeepEqual(oldAnnos, newAnnos) {
		_, err = op.client.CoreV1().Nodes().Update(node)
		if err != nil {
			klog.Errorf("failed to update node %s : %s", name, err.Error())
			return
		}
	}

	return
}

func (op *Operator) TaintNoExecuteNode(name string, taintsToAdd map[string]string, taintsToRemove []string) (err error) {
	taints := make([]v1.Taint, 0)
	noExecuteTaints := make([]v1.Taint, 0)
	if len(taintsToAdd) == 0 && len(taintsToRemove) == 0 {
		return
	}

	node, err := op.client.CoreV1().Nodes().Get(name, metav1.GetOptions{})
	if err != nil {
		return
	}

	for _, nodeTaint := range node.Spec.Taints {
		if nodeTaint.Effect == v1.TaintEffectNoExecute {
			noExecuteTaints = append(noExecuteTaints, nodeTaint)
		} else {
			taints = append(taints, nodeTaint)
		}
	}
	for key, value := range taintsToAdd {
		found := false
		for _, noExecuteTaint := range noExecuteTaints {
			if noExecuteTaint.Key == key {
				found = true
				break
			}
		}
		if !found {
			noExecuteTaints = append(noExecuteTaints, v1.Taint{
				Key:    key,
				Value:  value,
				Effect: v1.TaintEffectNoExecute,
			})
		}
	}
	for _, key := range taintsToRemove {
		for idx, noExecuteTaint := range noExecuteTaints {
			if noExecuteTaint.Key == key {
				noExecuteTaints = append(noExecuteTaints[:idx], noExecuteTaints[idx+1:]...)
			}
		}
	}

	_, err = op.client.CoreV1().Nodes().Update(node)
	if err != nil {
		klog.Errorf("failed to update node %s : %s", name, err.Error())
		return
	}

	return
}

func (op *Operator) DeletePvc(namespace string, name string) error {
	resource, err := op.k8sCache.GetResource(k8sModel.PersistentVolumeClaimKind, namespace, name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Warningf("pvc %s/%s is not found", namespace, name)
			return nil
		}
		klog.Errorf("failed to get pvc %s/%s : %s", namespace, name, err.Error())
		return err
	}

	return op.doDeletePvc(resource.(*k8sModel.PersistentVolumeClaim), false)
}

func (op *Operator) doDeletePvc(pvc *k8sModel.PersistentVolumeClaim, force bool) error {
	if !force && len(pvc.Labels) > 0 {
		selector := &metav1.LabelSelector{
			MatchLabels: pvc.Labels,
		}

		selectorStr, err := utils.ConvertLabelSelectorToStr(selector)
		if err != nil {
			klog.Errorf("failed to convert label selector: %s", err.Error())
			return err
		}

		statefulSets, err := op.k8sCache.ListStatefulSets(pvc.Namespace, selectorStr)
		if err != nil {
			klog.Errorf("failed to list stateful set : %s", err.Error())
			return err
		}
		if len(statefulSets) > 0 {
			statefulSetNames := make([]string, len(statefulSets))
			for _, statefulSet := range statefulSets {
				statefulSetNames = append(statefulSetNames, statefulSet.Namespace+"/"+statefulSet.Name)
			}
			err = fmt.Errorf("pvc %s/%s can not be deleted, it is still used by statefulsets %v", pvc.Namespace, pvc.Name, statefulSetNames)
			return err
		}
	}
	err := op.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("pvc %s/%s is not found ", pvc.Namespace, pvc.Name)
			return nil
		}
		klog.Errorf("failed to delete pvc %s/%s : %s", pvc.Namespace, pvc.Name, err.Error())
		return err
	}
	klog.Infof("succeed to delete pvc %s/%s", pvc.Namespace, pvc.Name)
	return nil
}

func (op *Operator) DeletePvcs(namespace string, labelSeletorStr string) error {
	pvcs, err := op.k8sCache.ListPersistentVolumeClaims(namespace, labelSeletorStr)
	if err != nil {
		klog.Errorf("failed to list pvcs : %s", err.Error())
		return err
	}
	for _, pvc := range pvcs {
		err := op.doDeletePvc(pvc, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (op *Operator) CreateSecret(namespace string, secretRequestBody *k8sModel.CreateSecretRequestBody) error {
	secret, err := buildSecret(namespace, secretRequestBody)
	if err != nil {
		return err
	}
	_, err = op.client.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		klog.Errorf("failed to create secret %s/%s : %s", namespace, secretRequestBody.Name, err.Error())
		return err
	}
	return nil
}

func (op *Operator) UpdateSecret(namespace string, walmSecret *k8sModel.CreateSecretRequestBody) (err error) {
	newSecret, err := buildSecret(namespace, walmSecret)
	if err != nil {
		return err
	}
	_, err = op.client.CoreV1().Secrets(namespace).Update(newSecret)
	if err != nil {
		klog.Errorf("failed to update secret : %s", err.Error())
		return
	}
	klog.Infof("succeed to update secret %s/%s", namespace, walmSecret.Name)
	return
}

func (op *Operator) DeleteSecret(namespace, name string) (err error) {
	err = op.client.CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("secret %s/%s is not found ", namespace, name)
			return nil
		}
		klog.Errorf("failed to delete secret : %s", err.Error())
		return
	}
	klog.Infof("succeed to delete secret %s/%s", namespace, name)
	return
}

func buildSecret(namespace string, walmSecret *k8sModel.CreateSecretRequestBody) (secret *v1.Secret, err error) {
	DataByte := make(map[string][]byte, 0)
	for k, v := range walmSecret.Data {
		DataByte[k], err = base64.StdEncoding.DecodeString(v)
		if err != nil {
			klog.Errorf("failed to decode secret : %+v %s", walmSecret.Data, err.Error())
			return
		}
	}
	klog.Infof("secret data: %+v", walmSecret.Data)
	secret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      walmSecret.Name,
		},
		Data: DataByte,
		Type: v1.SecretType(walmSecret.Type),
	}
	return
}

func buildService(namespace string, walmService *k8sModel.CreateServiceRequestBody) (service *v1.Service, err error) {
	var serviceType v1.ServiceType
	switch walmService.ServiceType {
	case "ClusterIP":
		serviceType = v1.ServiceTypeClusterIP
	case "NodePort":
		serviceType = v1.ServiceTypeNodePort
	case "LoadBalancer":
		serviceType = v1.ServiceTypeLoadBalancer
	case "ExternalName":
		serviceType = v1.ServiceTypeExternalName
	case "":
	default:
		return nil, errors.Errorf("invalid service type %s", walmService.ServiceType)
	}

	var servicePorts []v1.ServicePort
	for _, port := range walmService.Ports {
		var protocol v1.Protocol
		switch port.Protocol {
		case "TCP", "":
			protocol = v1.ProtocolTCP
		case "UDP":
			protocol = v1.ProtocolUDP
		case "SCTP":
			protocol = v1.ProtocolSCTP
		default:
			return nil, errors.Errorf("invalid service port protocol %s", port.Protocol)

		}

		servicePort := v1.ServicePort{
			Name:     port.Name,
			Protocol: protocol,
			Port:     port.Port,
			NodePort: port.NodePort,
		}
		if port.TargetPort != "" {
			targetPort, err := strconv.Atoi(port.TargetPort)
			if err != nil {
				return nil, errors.Errorf("targetPort not valid: %s", err.Error())
			}
			servicePort.TargetPort = intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: int32(targetPort),
			}
		}
		servicePorts = append(servicePorts, servicePort)
	}

	service = &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        walmService.Name,
			Namespace:   namespace,
			Labels:      walmService.Labels,
			Annotations: walmService.Annotations,
		},
		Spec: v1.ServiceSpec{
			Ports:       servicePorts,
			ClusterIP:   walmService.ClusterIp,
			Selector:    walmService.Selector,
			Type:        serviceType,
			ExternalIPs: walmService.ExternalIPs,
		},
	}
	return service, err
}

func (op *Operator) CreateService(namespace string, serviceRequestBody *k8sModel.CreateServiceRequestBody) error {
	service, err := buildService(namespace, serviceRequestBody)
	if err != nil {
		return err
	}

	_, err = op.client.CoreV1().Services(namespace).Create(service)
	if err != nil {
		klog.Errorf("failed to create service %s/%s : %s", namespace, serviceRequestBody.Name, err.Error())
		return err
	}
	klog.Infof("succeed to create service %s/%s", namespace, serviceRequestBody.Name)
	return nil
}

func (op *Operator) UpdateService(namespace string, serviceRequest *k8sModel.CreateServiceRequestBody, fullUpdate bool) error {
	service, err := op.k8sCache.GetResource(k8sModel.ServiceKind, namespace, serviceRequest.Name)
	if err != nil {
		if errorModel.IsNotFoundError(err) {
			klog.Errorf("service %s/%s not found", namespace, serviceRequest.Name)
			return errors.Errorf("service %s/%s not found", namespace, serviceRequest.Name)
		}
		klog.Errorf("failed to get old service %s/%s", namespace, serviceRequest.Name)
		return err
	}
	oldService := service.(*k8sModel.Service)
	oldK8sService, err := converter.ConvertServiceToK8s(oldService)
	if err != nil {
		return err
	}

	newK8sService, err := buildService(namespace, serviceRequest)
	if fullUpdate {
		if err != nil {
			return err
		}
		newK8sService.ResourceVersion = oldK8sService.ResourceVersion
		newK8sService.Spec.ClusterIP = oldK8sService.Spec.ClusterIP
		_, err = op.client.CoreV1().Services(namespace).Update(newK8sService)
		if err != nil {
			klog.Errorf("failed to update service %s/%s", namespace, serviceRequest.Name)
			return err
		}
	} else {
		newService, err := reuseServiceRequest(oldK8sService, newK8sService)
		if err != nil {
			return err
		}
		_, err = op.client.CoreV1().Services(namespace).Update(newService)
		if err != nil {
			klog.Errorf("failed to update service %s/%s", namespace, serviceRequest.Name)
			return err
		}
	}

	klog.Infof("succeed to update service %s/%s", namespace, serviceRequest.Name)
	return nil
}

func reuseServiceRequest(oldService *v1.Service, newService *v1.Service) (*v1.Service, error) {
	externalIPSet := make(map[string]bool)
	for _, externalIP := range oldService.Spec.ExternalIPs {
		externalIPSet[externalIP] = true
	}
	for _, externalIP := range newService.Spec.ExternalIPs {
		externalIPSet[externalIP] = true
	}
	var externalIps []string
	for k, _ := range externalIPSet {
		externalIps = append(externalIps, k)
	}

	servicePorts := make(map[string]v1.ServicePort)
	for _, port := range oldService.Spec.Ports {
		servicePorts[port.Name] = port
	}

	for _, port := range newService.Spec.Ports {
		servicePorts[port.Name] = port
	}

	var ports []v1.ServicePort
	for _, port := range servicePorts {
		ports = append(ports, port)
	}

	serviceType := oldService.Spec.Type
	if newService.Spec.Type != "" {
		serviceType = newService.Spec.Type
	}
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        oldService.Name,
			Namespace:   oldService.Namespace,
			ResourceVersion: oldService.ResourceVersion,
			Labels:      utils.MergeLabels(oldService.Labels, newService.Labels, nil),
			Annotations: utils.MergeLabels(oldService.Annotations, newService.Annotations, nil),
		},
		Spec: v1.ServiceSpec{
			Selector: utils.MergeLabels(oldService.Spec.Selector, newService.Spec.Selector, nil),
			ClusterIP: oldService.Spec.ClusterIP,
			ExternalIPs: externalIps,
			Ports: ports,
			Type: serviceType,
		},
	}
	return service, nil
}

func (op *Operator) DeleteService(namespace, name string) (err error) {
	err = op.client.CoreV1().Services(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if utils.IsK8sResourceNotFoundErr(err) {
			klog.Warningf("service %s/%s is not found ", namespace, name)
			return nil
		}
		klog.Errorf("failed to delete service : %s", err.Error())
		return err
	}
	klog.Infof("succeed to delete service %s/%s", namespace, name)
	return nil
}

func (op *Operator) UpdateIngress(namespace, ingressName string, requestBody *k8sModel.IngressRequestBody) (err error) {
	k8sIngress, err := op.client.ExtensionsV1beta1().Ingresses(namespace).Get(ingressName, metav1.GetOptions{})
	if err != nil {
		return
	}
	if len(requestBody.Annotations) != 0 {
		k8sIngress.Annotations = requestBody.Annotations
	}
	if len(k8sIngress.Spec.Rules) > 0 {
		rule := k8sIngress.Spec.Rules[0]
		if requestBody.Host != "" {
			k8sIngress.Spec.Rules[0].Host = requestBody.Host
		}
		if rule.HTTP != nil && len(rule.HTTP.Paths) > 0 {
			if requestBody.Path != "" {
				k8sIngress.Spec.Rules[0].HTTP.Paths[0].Path = requestBody.Path
			}
		}
	}
	if len(requestBody.Annotations) != 0 {
		k8sIngress.Annotations = requestBody.Annotations
	}
	_, err = op.client.ExtensionsV1beta1().Ingresses(namespace).Update(k8sIngress)
	if err != nil {
		klog.Errorf("failed to update ingress %s : %s", ingressName, err.Error())
		return
	}

	return
}

func (op *Operator) UpdateConfigMap(namespace, configMapName string, requestBody *k8sModel.ConfigMapRequestBody) (err error) {
	k8sConfigMap, err := op.client.CoreV1().ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		return
	}
	for key, value := range requestBody.Data {
		k8sConfigMap.Data[key] = value
	}
	_, err = op.client.CoreV1().ConfigMaps(namespace).Update(k8sConfigMap)
	if err != nil {
		klog.Errorf("failed to update configMap %s : %s", configMapName, err.Error())
		return
	}

	return
}

func (op *Operator) BackupAndUpdateReplicas(namespace, releaseName string, releaseStatus *k8sModel.ResourceSet, replicas int32) (err error) {
	releaseConfig, err := op.k8sReleaseConfigClient.TranswarpV1beta1().ReleaseConfigs(namespace).Get(releaseName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if releaseConfig.GetAnnotations() == nil {
		releaseConfig.SetAnnotations(map[string]string{})
	}

	for _, deployment := range releaseStatus.Deployments {
		k8sDeployment, err := op.client.AppsV1beta1().Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if k8sDeployment.Status.Replicas == replicas {
			return errors.Errorf("deployment %s replicas already be zero", k8sDeployment.Name)
		}

		releaseConfig.Annotations["deploy/" + k8sDeployment.Name] = strconv.FormatInt(int64(k8sDeployment.Status.Replicas), 10)
		k8sDeployment.Spec.Replicas = &replicas
		_, err = op.client.AppsV1beta1().Deployments(namespace).Update(k8sDeployment)
		if err != nil {
			return err
		}
	}

	for _, sts := range releaseStatus.StatefulSets {
		k8sStatefulSet, err := op.client.AppsV1beta1().StatefulSets(sts.Namespace).Get(sts.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if k8sStatefulSet.Status.Replicas == replicas {
			return errors.Errorf("statefulset %s replicas already be zero", k8sStatefulSet.Name)
		}
		releaseConfig.Annotations["sts/" + k8sStatefulSet.Name] = strconv.FormatInt(int64(k8sStatefulSet.Status.Replicas), 10)
		k8sStatefulSet.Spec.Replicas = &replicas
		_, err = op.client.AppsV1beta1().StatefulSets(namespace).Update(k8sStatefulSet)
		if err != nil {
			return err
		}
	}

	_, err = op.k8sReleaseConfigClient.TranswarpV1beta1().ReleaseConfigs(namespace).Update(releaseConfig)
	if err != nil {
		return err
	}
	return
}

func(op *Operator) RecoverReplicas(namespace string, releaseName string, releaseStatus *k8sModel.ResourceSet) error {
	releaseConfig, err := op.k8sReleaseConfigClient.TranswarpV1beta1().ReleaseConfigs(namespace).Get(releaseName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	deployList := map[string]string{}
	stsList := map[string]string{}
	for k, v := range releaseConfig.Annotations {
		if strings.HasPrefix(k, "sts/") {
			tokens := strings.Split(k, "/")
			stsList[tokens[1]] = v
			//delete(releaseConfig.Annotations, k)
		} else if strings.HasPrefix(k, "deploy/") {
			tokens := strings.Split(k, "/")
			deployList[tokens[1]] = v
			//delete(releaseConfig.Annotations, k)
		} else {
			continue
		}
	}

	// update deployment, sts
	for name, replicaStr := range deployList {
		deployment, err := op.client.AppsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		replicas64, err :=  strconv.ParseInt(replicaStr,10,32)
		if err != nil {
			return err
		}
		replicas32 := int32(replicas64)
		deployment.Spec.Replicas = &replicas32
		_, err = op.client.AppsV1beta1().Deployments(namespace).Update(deployment)
		if err != nil {
			return err
		}
	}

	for name, replicaStr := range stsList {
		sts, err := op.client.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		replicas64, err := strconv.ParseInt(replicaStr, 10, 32)
		replicas32 := int32(replicas64)
		sts.Spec.Replicas = &replicas32
		_, err = op.client.AppsV1beta1().StatefulSets(namespace).Update(sts)
		if err != nil {
			return err
		}
	}
	return nil
}


func NewOperator(client *kubernetes.Clientset, k8sCache k8s.Cache, kubeClients *helm.Client, k8sMigrationClient *migrationclientset.Clientset, k8sReleaseConfigClient *releaseconfigclientset.Clientset) *Operator {
	return &Operator{
		client:             client,
		k8sCache:           k8sCache,
		kubeClients:        kubeClients,
		k8sMigrationClient: k8sMigrationClient,
		k8sReleaseConfigClient: k8sReleaseConfigClient,
	}
}
