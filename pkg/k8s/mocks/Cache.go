// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	modelsk8s "WarpCloud/walm/pkg/models/k8s"

	mock "github.com/stretchr/testify/mock"

	release "WarpCloud/walm/pkg/models/release"

	tenant "WarpCloud/walm/pkg/models/tenant"
)

// Cache is an autogenerated mock type for the Cache type
type Cache struct {
	mock.Mock
}

// AddReleaseConfigHandler provides a mock function with given fields: OnAdd, OnUpdate, OnDelete
func (_m *Cache) AddReleaseConfigHandler(OnAdd func(interface{}), OnUpdate func(interface{}, interface{}), OnDelete func(interface{})) {
	_m.Called(OnAdd, OnUpdate, OnDelete)
}

// AddServiceHandler provides a mock function with given fields: OnAdd, OnUpdate, OnDelete
func (_m *Cache) AddServiceHandler(OnAdd func(interface{}), OnUpdate func(interface{}, interface{}), OnDelete func(interface{})) {
	_m.Called(OnAdd, OnUpdate, OnDelete)
}

// GetDeploymentEventList provides a mock function with given fields: namespace, name
func (_m *Cache) GetDeploymentEventList(namespace string, name string) (*modelsk8s.EventList, error) {
	ret := _m.Called(namespace, name)

	var r0 *modelsk8s.EventList
	if rf, ok := ret.Get(0).(func(string, string) *modelsk8s.EventList); ok {
		r0 = rf(namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*modelsk8s.EventList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNodeMigration provides a mock function with given fields: namespace, name
func (_m *Cache) GetNodeMigration(namespace string, name string) (*modelsk8s.MigList, error) {
	ret := _m.Called(namespace, name)

	var r0 *modelsk8s.MigList
	if rf, ok := ret.Get(0).(func(string, string) *modelsk8s.MigList); ok {
		r0 = rf(namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*modelsk8s.MigList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNodes provides a mock function with given fields: labelSelector
func (_m *Cache) GetNodes(labelSelector string) ([]*modelsk8s.Node, error) {
	ret := _m.Called(labelSelector)

	var r0 []*modelsk8s.Node
	if rf, ok := ret.Get(0).(func(string) []*modelsk8s.Node); ok {
		r0 = rf(labelSelector)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*modelsk8s.Node)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(labelSelector)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPodEventList provides a mock function with given fields: namespace, name
func (_m *Cache) GetPodEventList(namespace string, name string) (*modelsk8s.EventList, error) {
	ret := _m.Called(namespace, name)

	var r0 *modelsk8s.EventList
	if rf, ok := ret.Get(0).(func(string, string) *modelsk8s.EventList); ok {
		r0 = rf(namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*modelsk8s.EventList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPodLogs provides a mock function with given fields: namespace, podName, containerName, tailLines
func (_m *Cache) GetPodLogs(namespace string, podName string, containerName string, tailLines int64) (string, error) {
	ret := _m.Called(namespace, podName, containerName, tailLines)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string, string, int64) string); ok {
		r0 = rf(namespace, podName, containerName, tailLines)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string, int64) error); ok {
		r1 = rf(namespace, podName, containerName, tailLines)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetResource provides a mock function with given fields: kind, namespace, name
func (_m *Cache) GetResource(kind modelsk8s.ResourceKind, namespace string, name string) (modelsk8s.Resource, error) {
	ret := _m.Called(kind, namespace, name)

	var r0 modelsk8s.Resource
	if rf, ok := ret.Get(0).(func(modelsk8s.ResourceKind, string, string) modelsk8s.Resource); ok {
		r0 = rf(kind, namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(modelsk8s.Resource)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(modelsk8s.ResourceKind, string, string) error); ok {
		r1 = rf(kind, namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetResourceSet provides a mock function with given fields: releaseResourceMetas
func (_m *Cache) GetResourceSet(releaseResourceMetas []release.ReleaseResourceMeta) (*modelsk8s.ResourceSet, error) {
	ret := _m.Called(releaseResourceMetas)

	var r0 *modelsk8s.ResourceSet
	if rf, ok := ret.Get(0).(func([]release.ReleaseResourceMeta) *modelsk8s.ResourceSet); ok {
		r0 = rf(releaseResourceMetas)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*modelsk8s.ResourceSet)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]release.ReleaseResourceMeta) error); ok {
		r1 = rf(releaseResourceMetas)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStatefulSetEventList provides a mock function with given fields: namespace, name
func (_m *Cache) GetStatefulSetEventList(namespace string, name string) (*modelsk8s.EventList, error) {
	ret := _m.Called(namespace, name)

	var r0 *modelsk8s.EventList
	if rf, ok := ret.Get(0).(func(string, string) *modelsk8s.EventList); ok {
		r0 = rf(namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*modelsk8s.EventList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTenant provides a mock function with given fields: tenantName
func (_m *Cache) GetTenant(tenantName string) (*tenant.TenantInfo, error) {
	ret := _m.Called(tenantName)

	var r0 *tenant.TenantInfo
	if rf, ok := ret.Get(0).(func(string) *tenant.TenantInfo); ok {
		r0 = rf(tenantName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*tenant.TenantInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(tenantName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListMigrations provides a mock function with given fields: namespace, labelSelectorStr
func (_m *Cache) ListMigrations(namespace string, labelSelectorStr string) (*modelsk8s.MigList, error) {
	ret := _m.Called(namespace, labelSelectorStr)

	var r0 *modelsk8s.MigList
	if rf, ok := ret.Get(0).(func(string, string) *modelsk8s.MigList); ok {
		r0 = rf(namespace, labelSelectorStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*modelsk8s.MigList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, labelSelectorStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListPersistentVolumeClaims provides a mock function with given fields: namespace, labelSelectorStr
func (_m *Cache) ListPersistentVolumeClaims(namespace string, labelSelectorStr string) ([]*modelsk8s.PersistentVolumeClaim, error) {
	ret := _m.Called(namespace, labelSelectorStr)

	var r0 []*modelsk8s.PersistentVolumeClaim
	if rf, ok := ret.Get(0).(func(string, string) []*modelsk8s.PersistentVolumeClaim); ok {
		r0 = rf(namespace, labelSelectorStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*modelsk8s.PersistentVolumeClaim)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, labelSelectorStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListReleaseConfigs provides a mock function with given fields: namespace, labelSelectorStr
func (_m *Cache) ListReleaseConfigs(namespace string, labelSelectorStr string) ([]*modelsk8s.ReleaseConfig, error) {
	ret := _m.Called(namespace, labelSelectorStr)

	var r0 []*modelsk8s.ReleaseConfig
	if rf, ok := ret.Get(0).(func(string, string) []*modelsk8s.ReleaseConfig); ok {
		r0 = rf(namespace, labelSelectorStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*modelsk8s.ReleaseConfig)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, labelSelectorStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListSecrets provides a mock function with given fields: namespace, name
func (_m *Cache) ListSecrets(namespace string, name string) (*modelsk8s.SecretList, error) {
	ret := _m.Called(namespace, name)

	var r0 *modelsk8s.SecretList
	if rf, ok := ret.Get(0).(func(string, string) *modelsk8s.SecretList); ok {
		r0 = rf(namespace, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*modelsk8s.SecretList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListServices provides a mock function with given fields: namespace, labelSelectorStr
func (_m *Cache) ListServices(namespace string, labelSelectorStr string) ([]*modelsk8s.Service, error) {
	ret := _m.Called(namespace, labelSelectorStr)

	var r0 []*modelsk8s.Service
	if rf, ok := ret.Get(0).(func(string, string) []*modelsk8s.Service); ok {
		r0 = rf(namespace, labelSelectorStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*modelsk8s.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, labelSelectorStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListStatefulSets provides a mock function with given fields: namespace, labelSelectorStr
func (_m *Cache) ListStatefulSets(namespace string, labelSelectorStr string) ([]*modelsk8s.StatefulSet, error) {
	ret := _m.Called(namespace, labelSelectorStr)

	var r0 []*modelsk8s.StatefulSet
	if rf, ok := ret.Get(0).(func(string, string) []*modelsk8s.StatefulSet); ok {
		r0 = rf(namespace, labelSelectorStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*modelsk8s.StatefulSet)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, labelSelectorStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListStorageClasses provides a mock function with given fields: namespace, labelSelectorStr
func (_m *Cache) ListStorageClasses(namespace string, labelSelectorStr string) ([]*modelsk8s.StorageClass, error) {
	ret := _m.Called(namespace, labelSelectorStr)

	var r0 []*modelsk8s.StorageClass
	if rf, ok := ret.Get(0).(func(string, string) []*modelsk8s.StorageClass); ok {
		r0 = rf(namespace, labelSelectorStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*modelsk8s.StorageClass)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(namespace, labelSelectorStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListTenants provides a mock function with given fields: labelSelectorStr
func (_m *Cache) ListTenants(labelSelectorStr string) (*tenant.TenantInfoList, error) {
	ret := _m.Called(labelSelectorStr)

	var r0 *tenant.TenantInfoList
	if rf, ok := ret.Get(0).(func(string) *tenant.TenantInfoList); ok {
		r0 = rf(labelSelectorStr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*tenant.TenantInfoList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(labelSelectorStr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
