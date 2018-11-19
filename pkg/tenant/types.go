package tenant

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TenantInfoList struct {
	Items []*TenantInfo `json:"items" description:"tenant list"`
}

//Tenant Info
type TenantInfo struct {
	TenantName         string            `json:"tenant_name" description:"name of the tenant"`
	TenantCreationTime v1.Time           `json:"tenant_creation_time" description:"create time of the tenant"`
	TenantLabels       map[string]string `json:"tenant_labels"  description:"labels of the tenant"`
	TenantAnnotitions  map[string]string `json:"tenant_annotations"  description:"annotations of the tenant"`
	TenantStatus       string            `json:"tenant_status" description:"status of the tenant"`
	TenantQuotas       []*TenantQuota    `json:"tenant_quotas" description:"quotas of the tenant"`
	MultiTenant        bool              `json:"" `
	Ready              bool              `json:"ready" description:"tenant ready status"`
}

//Tenant Params Info
type TenantParams struct {
	TenantAnnotitions map[string]string    `json:"tenant_annotations"  description:"annotations of the tenant"`
	TenantLabels      map[string]string    `json:"tenant_labels"  description:"labels of the tenant"`
	TenantQuotas      []*TenantQuotaParams `json:"tenant_quotas" description:"quotas of the tenant"`
}

type TenantQuotaParams struct {
	QuotaName string           `json:"quota_name" description:"quota name"`
	Hard      *TenantQuotaInfo `json:"hard" description:"quota hard limit"`
}

type TenantQuota struct {
	QuotaName string           `json:"quota_name" description:"quota name"`
	Hard      *TenantQuotaInfo `json:"hard" description:"quota hard limit"`
	Used      *TenantQuotaInfo `json:"used" description:"quota used"`
}

//Quota Info
type TenantQuotaInfo struct {
	LimitCpu        string `json:"limit_cpu"  description:"requests of the CPU"`
	LimitMemory     string `json:"limit_memory"  description:"limit of the memory"`
	RequestsCPU     string `json:"requests_cpu"  description:"requests of the CPU"`
	RequestsMemory  string `json:"requests_memory"  description:"requests of the memory"`
	RequestsStorage string `json:"requests_storage"  description:"requests of the storage"`
	Pods            string `json:"pods" description:"num of the pods"`
}

/*
//Pod event Info
type PodEventInfo struct {
	FirstTimestamp time.Time `json:"first_timestamp" description:"first_timestamp of event"`
	LastTimestamp  time.Time `json:"last_timestamp" description:"last_timestamp of event"`
	Count          int       `json:"count" description:"count of event"`
	Type           string    `json:"type" description:"type of event"`
	Reason         string    `json:"reason" description:"reason of event"`
	Message        string    `json:"message" description:"message of event"`
}

//Pod log Info
type PodLogInfo struct {
	ContainerName string `json:"container_name" description:"name of container"`
	Log           string `json:"log" description:"log info"`
}

//Pod's events and log Info
type PodDetailInfo struct {
	Events []PodEventInfo `json:"events" description:"events info"`
	Log    []PodLogInfo   `json:"log" description:"logs info"`
}

//Service List for Tenant Context
type ServiceForTenantInfo struct {
	ApplicationType string `json:"application_type" description:"application_type of service"`
	ServiceStatus   string `json:"service_status" description:"service_status of service"`
	ServiceName     string `json:"service_name" description:"service_name of service"`
	ServiceHostname string `json:"service_hostname" description:"service_hostname of service"`
	Path            string `json:"proxy" description:"path of service"`
	Port            int    `json:"port" description:"port of service"`
}
*/
