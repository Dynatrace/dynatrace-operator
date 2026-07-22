// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"context"
	"fmt"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DynaKubeStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeStatus struct { //nolint:revive

	// Observed state of OneAgent
	// +kubebuilder:validation:Optional
	OneAgent oneagent.Status `json:"oneAgent,omitzero"`

	// Observed state of ActiveGate
	// +kubebuilder:validation:Optional
	ActiveGate activegate.Status `json:"activeGate,omitzero"`

	// Observed state of KubernetesMonitoring
	// +optional
	KubernetesMonitoring kubemon.Status `json:"kubernetesMonitoring,omitzero"`

	// Observed state of Code Modules
	// +kubebuilder:validation:Optional
	CodeModules oneagent.CodeModulesStatus `json:"codeModules,omitzero"`

	// Observed state of Metadata-Enrichment
	// +kubebuilder:validation:Optional
	MetadataEnrichment metadataenrichment.Status `json:"metadataEnrichment,omitzero"`

	// Observed state of KSPM
	// +kubebuilder:validation:Optional
	KSPM kspm.Status `json:"kspm,omitzero"`

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	// +kubebuilder:validation:Optional
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitzero"`

	// ProxyURLHash is the hashed value of what is in spec.proxy.
	// Used for setting it as an annotation value for components that use the proxy.
	// This annotation will cause the component to be restarted if the proxy changes.
	ProxyURLHash string `json:"proxyURLHash,omitempty"`

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

	// +kubebuilder:validation:Optional
	APIToken APITokenStatus `json:"apiToken,omitzero"`
}

type APITokenStatus struct {
	// +kubebuilder:validation:Optional
	AvailableOptionalScopes AvailableOptionalScopes `json:"availableOptionalScopes,omitzero"`

	// Platform indicates whether the provided apiToken is a platform token.
	Platform *bool `json:"platform,omitempty"`
}

type AvailableOptionalScopes struct {
	SettingsRead  *bool `json:"settingsRead,omitempty"`
	SettingsWrite *bool `json:"settingsWrite,omitempty"`
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
	_, log := logd.NewFromContext(ctx, "v1beta6")
	dk.Status.UpdatedTimestamp = metav1.Now()
	err := client.Status().Update(ctx, dk)

	if err != nil && k8serrors.IsConflict(err) {
		log.Info("could not update dynakube due to conflict")

		return nil
	}

	return errors.WithStack(err)
}
