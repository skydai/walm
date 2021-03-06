/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kube // import "helm.sh/helm/pkg/kube"

import (
	"context"
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"
	"io"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"log"
	"strings"
	"time"
	"transwarp/application-instance/pkg/apis/transwarp/v1beta1"
	instanceclientset "transwarp/application-instance/pkg/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"
	watchtools "k8s.io/client-go/tools/watch"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"regexp"
)

const resourceUpgradePolicyAnno = "helm.sh/upgrade-policy"
const upgradePolicy = "keep"

var stsUpgradeForbidMsgPattern = regexp.MustCompile("updates to statefulset spec for fields other than .+ are forbidden")

// ErrNoObjectsVisited indicates that during a visit operation, no matching objects were found.
var ErrNoObjectsVisited = errors.New("no objects visited")

// Client represents a client capable of communicating with the Kubernetes API.
type Client struct {
	Factory           Factory
	k8sInstanceClient *instanceclientset.Clientset
	Log               func(string, ...interface{})
	getter            genericclioptions.RESTClientGetter
}

// New creates a new Client.
func New(getter genericclioptions.RESTClientGetter, k8sInstanceClient *instanceclientset.Clientset) *Client {
	if getter == nil {
		getter = genericclioptions.NewConfigFlags(true)
	}
	// Add CRDs to the scheme. They are missing by default.
	if err := apiextv1beta1.AddToScheme(scheme.Scheme); err != nil {
		// This should never happen.
		panic(err)
	}
	return &Client{
		Factory:           cmdutil.NewFactory(getter),
		Log:               nopLogger,
		k8sInstanceClient: k8sInstanceClient,
		getter:            getter,
	}
}

var nopLogger = func(_ string, _ ...interface{}) {}

// Test connectivity to the Client
func (c *Client) IsReachable() error {
	client, _ := c.Factory.KubernetesClientSet()
	_, err := client.ServerVersion()
	if err != nil {
		return errors.New("Kubernetes cluster unreachable")
	}
	return nil
}

// Create creates Kubernetes resources specified in the resource list.
func (c *Client) Create(resources ResourceList) (*Result, error) {
	c.Log("creating %d resource(s)", len(resources))
	if err := perform(resources, createResource); err != nil {
		return nil, err
	}
	return &Result{Created: resources}, nil
}

// Wait up to the given timeout for the specified resources to be ready
func (c *Client) Wait(resources ResourceList, timeout time.Duration) error {
	cs, err := c.Factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	w := waiter{
		c:       cs,
		log:     c.Log,
		timeout: timeout,
	}
	return w.waitForResources(resources)
}

func (c *Client) namespace() string {
	if ns, _, err := c.Factory.ToRawKubeConfigLoader().Namespace(); err == nil {
		return ns
	}
	return v1.NamespaceDefault
}

// newBuilder returns a new resource builder for structured api objects.
func (c *Client) newBuilder() *resource.Builder {
	return c.Factory.NewBuilder().
		ContinueOnError().
		NamespaceParam(c.namespace()).
		DefaultNamespace().
		Flatten()
}

// Build validates for Kubernetes objects and returns unstructured infos.
func (c *Client) Build(reader io.Reader) (ResourceList, error) {
	result, err := c.newBuilder().
		Unstructured().
		Stream(reader, "").
		Do().Infos()
	return result, scrubValidationError(err)
}

// Update reads in the current configuration and a target configuration from io.reader
// and creates resources that don't already exists, updates resources that have been modified
// in the target configuration and deletes resources from the current configuration that are
// not present in the target configuration.
func (c *Client) Update(original, target ResourceList, force bool) (*Result, error) {
	updateErrors := []string{}
	res := &Result{}

	if instanceInfo := findInstanceInOriginal(original); instanceInfo != nil {
		c.Log("delete instance and instance modules")
		// Get Instance
		if c.k8sInstanceClient == nil {
			klog.Errorf("get k8sInstanceClient error")
			return nil, errors.Errorf("get k8sInstanceClient error, k8sInstanceClient is nil")
		}
		instance, err := c.k8sInstanceClient.TranswarpV1beta1().ApplicationInstances(instanceInfo.Namespace).Get(instanceInfo.Name, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("get instance error: %s", err.Error())
			return nil, err
		}
		// Delete Instance
		if err := deleteResource(instanceInfo); err != nil {
			klog.Errorf("delete instance error: %s", err.Error())
			return nil, err
		}
		res.Deleted = append(res.Deleted, instanceInfo)
		// Delete Instance Modules
		if err := c.deleteInstanceModules(instance); err != nil {
			return nil, err
		}
	}

	c.Log("checking %d resources for changes", len(target))
	err := target.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}

		helper := resource.NewHelper(info.Client, info.Mapping)
		if _, err := helper.Get(info.Namespace, info.Name, info.Export); err != nil {
			if !apierrors.IsNotFound(err) {
				return errors.Wrap(err, "could not get information about the resource")
			}

			// Since the resource does not exist, create it.
			if err := createResource(info); err != nil {
				return errors.Wrap(err, "failed to create resource")
			}

			// Append the created resource to the results
			res.Created = append(res.Created, info)

			kind := info.Mapping.GroupVersionKind.Kind
			c.Log("Created a new %s called %q\n", kind, info.Name)
			return nil
		}

		originalInfo := original.Get(info)
		if originalInfo == nil {
			c.Log("Warning: %s/%s is found in k8s, but not found in previous release info", info.Namespace, info.Name)
			err = c.RecreateResourceInfo(info)
			if err != nil {
				c.Log("Failed to recreate %q, err : %s", info.Name, err)
				return errors.Wrap(err, "Failed to recreate resource")
			}
			res.Updated = append(res.Updated, info)
			return nil
		}

		if err := updateResource(c, info, originalInfo.Object, force); err != nil {
			if stsUpgradeForbidden(err.Error()) {
				c.Log("Warning: %s", err.Error())
				resBytes, _ := json.MarshalIndent(originalInfo.Object, "", "  ")
				c.Log("old resource %s/%s manifest: \n%s", info.Namespace, info.Name, resBytes)
				err = c.RecreateResourceInfo(info)
				if err != nil {
					c.Log("Failed to recreate %q, err : %s", info.Name, err)
					updateErrors = append(updateErrors, errors.Wrap(err, "Failed to recreate resource").Error())
				}
			} else {
				c.Log("error updating the resource %q:\n\t %v", info.Name, err)
				updateErrors = append(updateErrors, err.Error())
			}
		}
		// Because we check for errors later, append the info regardless
		res.Updated = append(res.Updated, info)

		return nil
	})

	switch {
	case err != nil:
		return nil, err
	case len(updateErrors) != 0:
		return nil, errors.Errorf(strings.Join(updateErrors, " && "))
	}

	for _, info := range original.Difference(target) {
		c.Log("Deleting %q in %s...", info.Name, info.Namespace)
		if err := deleteResource(info); err != nil {
			c.Log("Failed to delete %q, err: %s", info.Name, err)
		} else {
			// Only append ones we succeeded in deleting
			res.Deleted = append(res.Deleted, info)
		}
	}
	return res, nil
}

func stsUpgradeForbidden(msg string) bool{
	return stsUpgradeForbidMsgPattern.MatchString(msg)
}

func (c *Client) RecreateResourceInfo(info *resource.Info) error {
	c.Log("Recreating %q in %s...", info.Name, info.Namespace)
	if err := deleteResource(info); err != nil {
		c.Log("Failed to delete %q, err: %s", info.Name, err)
		return errors.Wrap(err, "Failed to delete resource")
	}
	if err := createResource(info); err != nil {
		c.Log("Failed to create %q, err: %s", info.Name, err)
		return errors.Wrap(err, "failed to create resource")
	}
	return nil
}

// Delete deletes Kubernetes resources specified in the resources list. It will
// attempt to delete all resources even if one or more fail and collect any
// errors. All successfully deleted items will be returned in the `Deleted`
// ResourceList that is part of the result.
func (c *Client) Delete(resources ResourceList) (*Result, []error) {
	var errs []error
	res := &Result{}
	err := perform(resources, func(info *resource.Info) error {
		c.Log("Starting delete for %q %s", info.Name, info.Mapping.GroupVersionKind.Kind)
		if err := c.skipIfNotFound(deleteResource(info)); err != nil {
			// Collect the error and continue on
			errs = append(errs, err)
		} else {
			res.Deleted = append(res.Deleted, info)
		}
		return nil
	})
	if err != nil {
		// Rewrite the message from "no objects visited" if that is what we got
		// back
		if err == ErrNoObjectsVisited {
			err = errors.New("object not found, skipping delete")
		}
		errs = append(errs, err)
	}
	if errs != nil {
		return nil, errs
	}
	return res, nil
}

func (c *Client) skipIfNotFound(err error) error {
	if apierrors.IsNotFound(err) {
		c.Log("%v", err)
		return nil
	}
	return err
}

func (c *Client) watchTimeout(t time.Duration) func(*resource.Info) error {
	return func(info *resource.Info) error {
		return c.watchUntilReady(t, info)
	}
}

// WatchUntilReady watches the resources given and waits until it is ready.
//
// This function is mainly for hook implementations. It watches for a resource to
// hit a particular milestone. The milestone depends on the Kind.
//
// For most kinds, it checks to see if the resource is marked as Added or Modified
// by the Kubernetes event stream. For some kinds, it does more:
//
// - Jobs: A job is marked "Ready" when it has successfully completed. This is
//   ascertained by watching the Status fields in a job's output.
// - Pods: A pod is marked "Ready" when it has successfully completed. This is
//   ascertained by watching the status.phase field in a pod's output.
//
// Handling for other kinds will be added as necessary.
func (c *Client) WatchUntilReady(resources ResourceList, timeout time.Duration) error {
	// For jobs, there's also the option to do poll c.Jobs(namespace).Get():
	// https://github.com/adamreese/kubernetes/blob/master/test/e2e/job.go#L291-L300
	return perform(resources, c.watchTimeout(timeout))
}

func perform(infos ResourceList, fn func(*resource.Info) error) error {
	if len(infos) == 0 {
		return ErrNoObjectsVisited
	}

	for _, info := range infos {
		if err := fn(info); err != nil {
			return err
		}
	}
	return nil
}

func createResource(info *resource.Info) error {
	obj, err := resource.NewHelper(info.Client, info.Mapping).Create(info.Namespace, true, info.Object, nil)
	if err != nil {
		return err
	}
	return info.Refresh(obj, true)
}

func deleteResource(info *resource.Info) error {
	policy := metav1.DeletePropagationBackground
	opts := &metav1.DeleteOptions{PropagationPolicy: &policy}
	_, err := resource.NewHelper(info.Client, info.Mapping).DeleteWithOptions(info.Namespace, info.Name, opts)
	return err
}

func createPatch(target *resource.Info, current runtime.Object) ([]byte, types.PatchType, error) {
	oldData, err := json.Marshal(current)
	if err != nil {
		return nil, types.StrategicMergePatchType, errors.Wrap(err, "serializing current configuration")
	}
	newData, err := json.Marshal(target.Object)
	if err != nil {
		return nil, types.StrategicMergePatchType, errors.Wrap(err, "serializing target configuration")
	}
	if targetUnstructured, ok := target.Object.(*unstructured.Unstructured); ok {
		targetAnnotations := targetUnstructured.GetAnnotations()
		upgradePolicyAnno, ok := targetAnnotations[resourceUpgradePolicyAnno]
		if ok && upgradePolicyAnno == upgradePolicy {
			return nil, types.StrategicMergePatchType, nil
		}
	}

	// Fetch the current object for the three way merge
	helper := resource.NewHelper(target.Client, target.Mapping)
	currentObj, err := helper.Get(target.Namespace, target.Name, target.Export)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, types.StrategicMergePatchType, errors.Wrapf(err, "unable to get data for current object %s/%s", target.Namespace, target.Name)
	}

	// Even if currentObj is nil (because it was not found), it will marshal just fine
	currentData, err := json.Marshal(currentObj)
	if err != nil {
		return nil, types.StrategicMergePatchType, errors.Wrap(err, "serializing live configuration")
	}

	// Get a versioned object
	versionedObject := AsVersioned(target)

	// Unstructured objects, such as CRDs, may not have an not registered error
	// returned from ConvertToVersion. Anything that's unstructured should
	// use the jsonpatch.CreateMergePatch. Strategic Merge Patch is not supported
	// on objects like CRDs.
	if _, ok := versionedObject.(runtime.Unstructured); ok {
		// fall back to generic JSON merge patch
		patch, err := jsonpatch.CreateMergePatch(oldData, newData)
		return patch, types.MergePatchType, err
	}

	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(versionedObject)
	if err != nil {
		return nil, types.StrategicMergePatchType, errors.Wrap(err, "unable to create patch metadata from object")
	}

	patch, err := strategicpatch.CreateThreeWayMergePatch(oldData, newData, currentData, patchMeta, true)
	return patch, types.StrategicMergePatchType, err
}

func updateResource(c *Client, target *resource.Info, currentObj runtime.Object, force bool) error {
	patch, patchType, err := createPatch(target, currentObj)
	if err != nil {
		return errors.Wrap(err, "failed to create patch")
	}
	if patch == nil {
		c.Log("Looks like there are no changes for %s %q", target.Mapping.GroupVersionKind.Kind, target.Name)
		// This needs to happen to make sure that tiller has the latest info from the API
		// Otherwise there will be no labels and other functions that use labels will panic
		if err := target.Get(); err != nil {
			return errors.Wrap(err, "error trying to refresh resource information")
		}
	} else {
		// send patch to server
		helper := resource.NewHelper(target.Client, target.Mapping)

		obj, err := helper.Patch(target.Namespace, target.Name, patchType, patch, nil)
		if err != nil {
			kind := target.Mapping.GroupVersionKind.Kind
			log.Printf("Cannot patch %s: %q (%v)", kind, target.Name, err)

			if force {
				// Attempt to delete...
				if err := deleteResource(target); err != nil {
					return err
				}
				log.Printf("Deleted %s: %q", kind, target.Name)

				// ... and recreate
				if err := createResource(target); err != nil {
					return errors.Wrap(err, "failed to recreate resource")
				}
				log.Printf("Created a new %s called %q\n", kind, target.Name)

				// No need to refresh the target, as we recreated the resource based
				// on it. In addition, it might not exist yet and a call to `Refresh`
				// may fail.
			} else {
				log.Print("Use --force to force recreation of the resource")
				return err
			}
		} else {
			// When patch succeeds without needing to recreate, refresh target.
			target.Refresh(obj, true)
		}
	}

	return nil
}

func (c *Client) watchUntilReady(timeout time.Duration, info *resource.Info) error {
	kind := info.Mapping.GroupVersionKind.Kind
	switch kind {
	case "Job", "Pod":
	default:
		return nil
	}

	c.Log("Watching for changes to %s %s with timeout of %v", kind, info.Name, timeout)

	w, err := resource.NewHelper(info.Client, info.Mapping).WatchSingle(info.Namespace, info.Name, info.ResourceVersion)
	if err != nil {
		return err
	}

	// What we watch for depends on the Kind.
	// - For a Job, we watch for completion.
	// - For all else, we watch until Ready.
	// In the future, we might want to add some special logic for types
	// like Ingress, Volume, etc.

	ctx, cancel := watchtools.ContextWithOptionalTimeout(context.Background(), timeout)
	defer cancel()
	_, err = watchtools.UntilWithoutRetry(ctx, w, func(e watch.Event) (bool, error) {
		// Make sure the incoming object is versioned as we use unstructured
		// objects when we build manifests
		obj := convertWithMapper(e.Object, info.Mapping)
		switch e.Type {
		case watch.Added, watch.Modified:
			// For things like a secret or a config map, this is the best indicator
			// we get. We care mostly about jobs, where what we want to see is
			// the status go into a good state. For other types, like ReplicaSet
			// we don't really do anything to support these as hooks.
			c.Log("Add/Modify event for %s: %v", info.Name, e.Type)
			switch kind {
			case "Job":
				return c.waitForJob(obj, info.Name)
			case "Pod":
				return c.waitForPodSuccess(obj, info.Name)
			}
			return true, nil
		case watch.Deleted:
			c.Log("Deleted event for %s", info.Name)
			return true, nil
		case watch.Error:
			// Handle error and return with an error.
			c.Log("Error event for %s", info.Name)
			return true, errors.Errorf("failed to deploy %s", info.Name)
		default:
			return false, nil
		}
	})
	return err
}

// waitForJob is a helper that waits for a job to complete.
//
// This operates on an event returned from a watcher.
func (c *Client) waitForJob(obj runtime.Object, name string) (bool, error) {
	o, ok := obj.(*batch.Job)
	if !ok {
		return true, errors.Errorf("expected %s to be a *batch.Job, got %T", name, obj)
	}

	for _, c := range o.Status.Conditions {
		if c.Type == batch.JobComplete && c.Status == "True" {
			return true, nil
		} else if c.Type == batch.JobFailed && c.Status == "True" {
			return true, errors.Errorf("job failed: %s", c.Reason)
		}
	}

	c.Log("%s: Jobs active: %d, jobs failed: %d, jobs succeeded: %d", name, o.Status.Active, o.Status.Failed, o.Status.Succeeded)
	return false, nil
}

// waitForPodSuccess is a helper that waits for a pod to complete.
//
// This operates on an event returned from a watcher.
func (c *Client) waitForPodSuccess(obj runtime.Object, name string) (bool, error) {
	o, ok := obj.(*v1.Pod)
	if !ok {
		return true, errors.Errorf("expected %s to be a *v1.Pod, got %T", name, obj)
	}

	switch o.Status.Phase {
	case v1.PodSucceeded:
		fmt.Printf("Pod %s succeeded\n", o.Name)
		return true, nil
	case v1.PodFailed:
		return true, errors.Errorf("pod %s failed", o.Name)
	case v1.PodPending:
		fmt.Printf("Pod %s pending\n", o.Name)
	case v1.PodRunning:
		fmt.Printf("Pod %s running\n", o.Name)
	}

	return false, nil
}

// scrubValidationError removes kubectl info from the message.
func scrubValidationError(err error) error {
	if err == nil {
		return nil
	}
	const stopValidateMessage = "if you choose to ignore these errors, turn validation off with --validate=false"

	if strings.Contains(err.Error(), stopValidateMessage) {
		return errors.New(strings.ReplaceAll(err.Error(), "; "+stopValidateMessage, ""))
	}
	return err
}

// WaitAndGetCompletedPodPhase waits up to a timeout until a pod enters a completed phase
// and returns said phase (PodSucceeded or PodFailed qualify).
func (c *Client) WaitAndGetCompletedPodPhase(name string, timeout time.Duration) (v1.PodPhase, error) {
	client, _ := c.Factory.KubernetesClientSet()
	to := int64(timeout)
	watcher, err := client.CoreV1().Pods(c.namespace()).Watch(metav1.ListOptions{
		FieldSelector:  fmt.Sprintf("metadata.name=%s", name),
		TimeoutSeconds: &to,
	})

	for event := range watcher.ResultChan() {
		p, ok := event.Object.(*v1.Pod)
		if !ok {
			return v1.PodUnknown, fmt.Errorf("%s not a pod", name)
		}
		switch p.Status.Phase {
		case v1.PodFailed:
			return v1.PodFailed, nil
		case v1.PodSucceeded:
			return v1.PodSucceeded, nil
		}
	}

	return v1.PodUnknown, err
}

func (c *Client) deleteInstanceModules(instance *v1beta1.ApplicationInstance) error {

	client, err := c.Factory.KubernetesClientSet()
	if err != nil {
		klog.Errorf("get kubernetes clientSet err: %s", err.Error())
		return err
	}

	for _, module := range instance.Status.Modules {
		retryCount := 0
		retryLimit := 3
		for {
			err := deleteInstanceModule(client, module)
			// Handler Error
			if err != nil {
				if apierrors.IsNotFound(err) {
					klog.Infof("%s %s not found: %s", module.ResourceRef.Kind, module.ResourceRef.Name, err.Error())
					break
				} else if apierrors.IsConflict(err) {
					if retryCount == retryLimit {
						klog.Errorf("delete instance module %s %s timeout: %s", module.ResourceRef.Kind, module.ResourceRef.Name, err.Error())
						return err
					}
					time.Sleep(500 * time.Millisecond)
					retryCount++
					continue
				} else {
					klog.Errorf("delete instance module %s %s error: %s", module.ResourceRef.Kind, module.ResourceRef.Name, err.Error())
					return err
				}
			}
			break
		}

	}
	return nil
}

func deleteInstanceModule(client *kubernetes.Clientset, module v1beta1.ResourceReference) error {
	var err error
	switch module.ResourceRef.Kind {
	case "ConfigMap":
		err = client.CoreV1().ConfigMaps(module.ResourceRef.Namespace).Delete(module.ResourceRef.Name, &metav1.DeleteOptions{})
	case "Service":
		err = client.CoreV1().Services(module.ResourceRef.Namespace).Delete(module.ResourceRef.Name, &metav1.DeleteOptions{})
	case "StatefulSet":
		err = client.AppsV1beta1().StatefulSets(module.ResourceRef.Namespace).Delete(module.ResourceRef.Name, &metav1.DeleteOptions{})
	case "DaemonSet":
		err = client.ExtensionsV1beta1().DaemonSets(module.ResourceRef.Namespace).Delete(module.ResourceRef.Name, &metav1.DeleteOptions{})
	case "Deployment":
		err = client.ExtensionsV1beta1().Deployments(module.ResourceRef.Namespace).Delete(module.ResourceRef.Name, &metav1.DeleteOptions{})
	case "Ingress":
		err = client.ExtensionsV1beta1().Ingresses(module.ResourceRef.Namespace).Delete(module.ResourceRef.Name, &metav1.DeleteOptions{})
	case "Job":
		err = client.BatchV1().Jobs(module.ResourceRef.Namespace).Delete(module.ResourceRef.Name, &metav1.DeleteOptions{})
	case "Secret":
		err = client.CoreV1().Secrets(module.ResourceRef.Namespace).Delete(module.ResourceRef.Name, &metav1.DeleteOptions{})
	}
	if err != nil {
		return err
	}
	klog.Infof("instance module %s %s deleted", module.ResourceRef.Kind, module.ResourceRef.Name)
	return nil
}

func findInstanceInOriginal(original ResourceList) *resource.Info {
	for _, info := range original {
		if info.Object.GetObjectKind().GroupVersionKind().Kind == "ApplicationInstance" {
			return info
		}
	}
	return nil
}
