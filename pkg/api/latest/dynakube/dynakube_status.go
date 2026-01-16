package dynakube

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DynaKubeStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeStatus struct { //nolint:revive

	// Observed state of OneAgent
	OneAgent oneagent.Status `json:"oneAgent,omitempty"`

	// Observed state of ActiveGate
	ActiveGate activegate.Status `json:"activeGate,omitempty"`

	// Observed state of Code Modules
	CodeModules oneagent.CodeModulesStatus `json:"codeModules,omitempty"`

	// Observed state of Metadata-Enrichment
	MetadataEnrichment metadataenrichment.Status `json:"metadataEnrichment,omitempty"`

	// Observed state of Kspm
	Kspm kspm.Status `json:"kspm,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// ProxyURLHash is the hashed value of what is in spec.proxy.
	// Used for setting it as an annotation value for components that use the proxy.
	// This annotation will cause the component to be restarted if the proxy changes.
	ProxyURLHash string `json:"proxyURLHash,omitempty"`

	// Observed state of Dynatrace API
	DynatraceAPI DynatraceAPIStatus `json:"dynatraceApi,omitempty"`

	// Defines the current state (Running, Updating, Error, ...)
	Phase status.DeploymentPhase `json:"phase,omitempty"`

	// KubeSystemUUID contains the UUID of the current Kubernetes cluster
	KubeSystemUUID string `json:"kubeSystemUUID,omitempty"`

	// KubernetesClusterMEID contains the ID of the monitored entity that points to the Kubernetes cluster
	KubernetesClusterMEID string `json:"kubernetesClusterMEID,omitempty"`

	// KubernetesClusterName contains the display name (also know as label) of the monitored entity that points to the Kubernetes cluster
	KubernetesClusterName string `json:"kubernetesClusterName,omitempty"`

	// Conditions includes status about the current state of the instance
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type DynatraceAPIStatus struct {
	// Time of the last token request
	LastTokenScopeRequest metav1.Time `json:"lastTokenScopeRequest,omitempty"`

	LastRequestPeriod metav1.Time                  `json:"lastRequestPeriod,omitempty"`
	ClientConfig      map[string]string            `json:"clientConfig,omitempty"`
	Requests          map[string]map[string]string `json:"requests,omitempty"`
}

func (api *DynatraceAPIStatus) StartNewPeriod(currentClientConfig map[string]string, threshold time.Duration) {
	if !maps.Equal(api.ClientConfig, currentClientConfig) || api.LastRequestPeriod.IsZero() || time.Since(api.LastRequestPeriod.Time) > threshold {
		api.LastRequestPeriod = metav1.Now()
		api.Requests = make(map[string]map[string]string)
		api.ClientConfig = currentClientConfig
	}
}

func (api *DynatraceAPIStatus) IsRequestAllowed(apiName string, props map[string]string) bool {
	if api.Requests == nil {
		api.Requests = make(map[string]map[string]string)
	}

	prevProps, exists := api.Requests[apiName]

	return !exists || !maps.Equal(props, prevProps)
}

func (api *DynatraceAPIStatus) AddRequest(apiName string, props map[string]string) {
	if api.Requests == nil {
		api.Requests = make(map[string]map[string]string)
	}

	api.Requests[apiName] = props
}

func (api *DynatraceAPIStatus) RemoveRequest(apiName string) {
	delete(api.Requests, apiName)
}

func GetCacheValidMessage(functionName string, lastRequestTimestamp metav1.Time, timeout time.Duration) string {
	remaining := timeout - time.Since(lastRequestTimestamp.Time)

	return fmt.Sprintf("skipping %s, last request was made less than %d minutes ago, %d minutes remaining until next request",
		functionName,
		int(timeout.Minutes()),
		int(remaining.Minutes()))
}

// SetPhase sets the status phase on the DynaKube object.
func (dk *DynaKubeStatus) SetPhase(phase status.DeploymentPhase) bool {
	upd := phase != dk.Phase
	dk.Phase = phase

	return upd
}

func (dk *DynaKube) UpdateStatus(ctx context.Context, client client.Client) error {
	dk.Status.UpdatedTimestamp = metav1.Now()
	err := client.Status().Update(ctx, dk)

	if err != nil && k8serrors.IsConflict(err) {
		log.Info("could not update dynakube due to conflict", "name", dk.Name)

		return nil
	}

	return errors.WithStack(err)
}
