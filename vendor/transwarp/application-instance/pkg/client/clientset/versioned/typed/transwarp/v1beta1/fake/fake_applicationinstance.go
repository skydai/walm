/*
Copyright 2018 The Kubernetes Authors.

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

package fake

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1beta1 "transwarp/application-instance/pkg/apis/transwarp/v1beta1"
)

// FakeApplicationInstances implements ApplicationInstanceInterface
type FakeApplicationInstances struct {
	Fake *FakeTranswarpV1beta1
	ns   string
}

var applicationinstancesResource = schema.GroupVersionResource{Group: "transwarp.k8s.io", Version: "v1beta1", Resource: "applicationinstances"}

var applicationinstancesKind = schema.GroupVersionKind{Group: "transwarp.k8s.io", Version: "v1beta1", Kind: "ApplicationInstance"}

// Get takes name of the applicationInstance, and returns the corresponding applicationInstance object, and an error if there is any.
func (c *FakeApplicationInstances) Get(name string, options v1.GetOptions) (result *v1beta1.ApplicationInstance, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(applicationinstancesResource, c.ns, name), &v1beta1.ApplicationInstance{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ApplicationInstance), err
}

// List takes label and field selectors, and returns the list of ApplicationInstances that match those selectors.
func (c *FakeApplicationInstances) List(opts v1.ListOptions) (result *v1beta1.ApplicationInstanceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(applicationinstancesResource, applicationinstancesKind, c.ns, opts), &v1beta1.ApplicationInstanceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.ApplicationInstanceList{}
	for _, item := range obj.(*v1beta1.ApplicationInstanceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested applicationInstances.
func (c *FakeApplicationInstances) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(applicationinstancesResource, c.ns, opts))

}

// Create takes the representation of a applicationInstance and creates it.  Returns the server's representation of the applicationInstance, and an error, if there is any.
func (c *FakeApplicationInstances) Create(applicationInstance *v1beta1.ApplicationInstance) (result *v1beta1.ApplicationInstance, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(applicationinstancesResource, c.ns, applicationInstance), &v1beta1.ApplicationInstance{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ApplicationInstance), err
}

// Update takes the representation of a applicationInstance and updates it. Returns the server's representation of the applicationInstance, and an error, if there is any.
func (c *FakeApplicationInstances) Update(applicationInstance *v1beta1.ApplicationInstance) (result *v1beta1.ApplicationInstance, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(applicationinstancesResource, c.ns, applicationInstance), &v1beta1.ApplicationInstance{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ApplicationInstance), err
}

// Delete takes name of the applicationInstance and deletes it. Returns an error if one occurs.
func (c *FakeApplicationInstances) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(applicationinstancesResource, c.ns, name), &v1beta1.ApplicationInstance{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeApplicationInstances) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(applicationinstancesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1beta1.ApplicationInstanceList{})
	return err
}

// Patch applies the patch and returns the patched applicationInstance.
func (c *FakeApplicationInstances) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.ApplicationInstance, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(applicationinstancesResource, c.ns, name, data, subresources...), &v1beta1.ApplicationInstance{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ApplicationInstance), err
}