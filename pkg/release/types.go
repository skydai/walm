package release

type ReleaseInfoList struct {
	Items []*ReleaseInfo `json:"items" description:"releases list"`
}

type ReleaseInfo struct {
	Name            string                 `json:"name" description:"name of the release"`
	ConfigValues    map[string]interface{} `json:"configvalues" description:"extra values added to the chart"`
	Version         int32                  `json:"version" description:"version of the release"`
	Namespace       string                 `json:"namespace" description:"namespace of release"`
	Statuscode      int32                  `json:"statuscode" description:"status code of release"`
	Dependencies    map[string]string      `json:"dependencies" description:"map of dependency chart name and release"`
	ChartName       string                 `json:"chartname" description:"chart name"`
	ChartVersion    string                 `json:"chartversion" description:"chart version"`
	ChartAppVersion string                 `json:"chartappversion" description:"jsonnet app version"`
	Status          ReleaseStatus          `json:"releasestatus" description:"status of release"`
}

type ReleaseResource struct{
	Kind string
	Resource interface{}
}

type ReleaseResourceMeta struct {
	Kind string
	Namespace string
	Name string
}

type ChartValicationInfo struct {
	Name            string                 `json:"name" description:"name of the release"`
	ConfigValues    map[string]interface{} `json:"configvalues" description:"extra values added to the chart"`
	Version         int32                  `json:"version" description:"version of the release"`
	Namespace       string                 `json:"namespace" description:"namespace of release"`
	Dependencies    map[string]string      `json:"dependencies" description:"map of dependency chart name and release"`
	ChartName       string                 `json:"chartname" description:"chart name"`
	ChartVersion    string                 `json:"chartversion" description:"chart version"`
	RenderStatus    string           	   `json:"render_status" description:"status of rending "`
	RenderResult    map[string]string	   `json:"render_result" description:"result of rending "`
	DryRunStatus    string				   `json:"dryrun_status" description:"status of dry run "`
	DryRunResult    map[string]string	   `json:"dryrun_result" description:"result of dry run "`
	ErrorMessage    string                 `json:"error_message" description:" error msg "`
}

type ReleaseStatus struct {
	Resources []ReleaseResource
}

type ReleaseRequest struct {
	Name         string                 `json:"name" description:"name of the release"`
	Namespace    string                 `json:"namespace" description:"namespace of release"`
	ChartName    string                 `json:"chartname" description:"chart name"`
	ChartVersion string                 `json:"chartversion" description:"chart repo"`
	ConfigValues map[string]interface{} `json:"configvalues" description:"extra values added to the chart"`
	Dependencies map[string]string      `json:"dependencies" description:"map of dependency chart name and release"`
	//ChartURL string
}

type DependencyDeclare struct {
	// name of dependency declaration
	Name string `json:"name,omitempty"`
	// dependency variable mappings
	Requires map[string]string `json:"requires,omitempty"`
}

type AppDependency struct {
	Name string `json:"name,omitempty"`
	Dependencies []*DependencyDeclare `json:"dependencies"`
}

type HelmNativeValues struct {
	ChartName string `json:"chartName"`
	ChartVersion string `json:"chartVersion"`
	AppVersion string `json:"appVersion"`
	ReleaseName string `json:"releaseName"`
	ReleaseNamespace string `json:"releaseNamespace"`
}

type AppHelmValues struct {
	Dependencies []*DependencyDeclare `json:"dependencies"`
	NativeValues HelmNativeValues `json:"HelmNativeValues"`
}

type ProjectParams struct {
	CommonValues map[string]interface{} `json:"common_values" description:"common values added to the chart"`
	Releases []*ReleaseRequest `json:"releases" description:"list of release of the project"`
}

type ProjectInfo struct {
	Name string `json:"name" description:"project name"`
	CommonValues map[string]interface{} `json:"common_values" description:"common values added to the chart"`
	Releases []*ReleaseInfo `json:"releases" description:"list of release of the project"`
}

type ProjectInfoList struct {
	Items []*ProjectInfo `json:"items" description:"project info list"`
}

type HelmExtraLabels struct {
	ProjectName string `json:"projectname" description:"project name which belongs to"`
	HelmLabels map[string]interface{} `json:"helmlabels"`
}

type HelmValues struct {
	HelmExtraLabels *HelmExtraLabels `json:"HelmExtraLabels"`
	AppHelmValues *AppHelmValues `json:"HelmAdditionalValues"`
}
