/*
Copyright The Kubernetes Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1beta1

import (
	time "time"
	transwarpv1beta1 "transwarp/application-instance/pkg/apis/transwarp/v1beta1"
	versioned "transwarp/application-instance/pkg/client/clientset/versioned"
	internalinterfaces "transwarp/application-instance/pkg/client/informers/externalversions/internalinterfaces"
	v1beta1 "transwarp/application-instance/pkg/client/listers/transwarp/v1beta1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ApplicationInstanceInformer provides access to a shared informer and lister for
// ApplicationInstances.
type ApplicationInstanceInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1beta1.ApplicationInstanceLister
}

type applicationInstanceInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewApplicationInstanceInformer constructs a new informer for ApplicationInstance type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewApplicationInstanceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredApplicationInstanceInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredApplicationInstanceInformer constructs a new informer for ApplicationInstance type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredApplicationInstanceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TranswarpV1beta1().ApplicationInstances(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TranswarpV1beta1().ApplicationInstances(namespace).Watch(options)
			},
		},
		&transwarpv1beta1.ApplicationInstance{},
		resyncPeriod,
		indexers,
	)
}

func (f *applicationInstanceInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredApplicationInstanceInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *applicationInstanceInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&transwarpv1beta1.ApplicationInstance{}, f.defaultInformer)
}

func (f *applicationInstanceInformer) Lister() v1beta1.ApplicationInstanceLister {
	return v1beta1.NewApplicationInstanceLister(f.Informer().GetIndexer())
}
