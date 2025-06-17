package dynakube

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
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
	MetadataEnrichment MetadataEnrichmentStatus `json:"metadataEnrichment,omitempty"`

	// Observed state of Kspm
	Kspm kspm.Status `json:"kspm,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Observed state of Dynatrace API
	// +kubebuilder:printcolumn:name="dynatraceApi",type=string,JSONPath=`.spec.dynatraceApi`
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
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type DynatraceAPIStatus struct {
	// Time of the last token request
	LastTokenScopeRequest metav1.Time `json:"lastTokenScopeRequest,omitempty"`
}

func GetCacheValidMessage(functionName string, lastRequestTimestamp metav1.Time, timeout time.Duration) string {
	remaining := timeout - time.Since(lastRequestTimestamp.Time)

	return fmt.Sprintf("skipping %s, last request was made less than %d minutes ago, %d minutes remaining until next request",
		functionName,
		int(timeout.Minutes()),
		int(remaining.Minutes()))
}

type EnrichmentRuleType string

const (
	EnrichmentLabelRule          EnrichmentRuleType = "LABEL"
	EnrichmentAnnotationRule     EnrichmentRuleType = "ANNOTATION"
	MetadataAnnotation           string             = "metadata.dynatrace.com"
	MetadataPrefix               string             = MetadataAnnotation + "/"
	enrichmentNamespaceKeyPrefix string             = "k8s.namespace."
)

type MetadataEnrichmentStatus struct {
	Rules []EnrichmentRule `json:"rules,omitempty"`
}

type EnrichmentRule struct {
	Type   EnrichmentRuleType `json:"type,omitempty"`
	Source string             `json:"source,omitempty"`
	Target string             `json:"target,omitempty"`
}

func (rule EnrichmentRule) ToAnnotationKey() string {
	if rule.Target == "" {
		return ""
	}

	return MetadataPrefix + rule.Target
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

func GetEmptyTargetEnrichmentKey(metadataType, key string) string {
	return enrichmentNamespaceKeyPrefix + strings.ToLower(metadataType) + "." + key
}
