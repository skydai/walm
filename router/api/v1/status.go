/*
 * walm
 *
 * warp application lifecycle manager
 *
 * API version: 1.0.0
 * Contact: bing.han@transwarp.io
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package v1

type Status struct {
	Name      string `json:"name" description:"name of the release"`
	Revision  string `json:"revision" description:"revision of the release"`
	Updated   string `json:"updated"  description:"last updated datetime of the release"`
	Status    string `json:"status,omitempty" description:"status of the release"`
	Chart     string `json:"chart,omitempty"  description:"chart of the release"`
	Namespace string `json:"namespace"  description:"namespace of the release"`
}
