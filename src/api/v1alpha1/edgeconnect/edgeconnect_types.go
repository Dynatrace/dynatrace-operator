// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1alpha1
package v1alpha1

import (
	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	"github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EdgeConnectSpec defines the desired state of EdgeConnect
type EdgeConnectSpec struct {
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// ApiServer location of the Dynatrace API to connect to, including your specific environment UUID
	// +kubebuilder:validation:Required
	ApiServer string `json:"apiServer"`

	// Oauth authorization configuration
	// +kubebuilder:validation:Required
	Oauth OAuthSpec `json:"oauth,omitempty"`

	// Optional: set host restrictions
	// +kubebuilder:validation:Optional
	HostRestrictions string `json:"hostRestrictions"`

	// ImageRef image reference
	ImageRef ImageRefSpec `json:"imageRef,omitempty"`

	// AutoUpdate auto update
	AutoUpdate bool `json:"autoUpdate,omitempty"`

	// Optional: Pull secret for your private registry
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// Optional: Adds additional annotations for the EdgeConnect pods
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Optional: Adds additional labels for the EdgeConnect pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Optional: List of environment variables to set for the EdgeConnect
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Amount of replicas for your DynaKube
	Replicas *int32 `json:"replicas,omitempty"`

	// Optional: define resources requests and limits for single pods
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Node selector to control the selection of nodes (optional)
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Optional: set tolerations for the EdgeConnect pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Optional: set topology spread constraints for the EdgeConnect pods
	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

type OAuthSpec struct {
	// Credentials for the EdgeConnect to connect back to Dynatrace.
	// +kubebuilder:validation:Required
	ClientSecret string `json:"clientSecret,omitempty"`
	// Optional: endpoint for the EdgeConnect to connect to
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint,omitempty"`
	// Optional: resource
	// +kubebuilder:validation:Optional
	Resource string `json:"resource,omitempty"`
}

type ImageRefSpec struct {
	// Optional: If specified, indicates the EdgeConnect repository to use
	// +kubebuilder:validation:Optional
	Repository string `json:"repository,omitempty"`

	// Optional: tag
	// +kubebuilder:validation:Optional
	Tag string `json:"tag,omitempty"`
}

// EdgeConnectStatus defines the observed state of DynaKube
type EdgeConnectStatus struct {
	// Defines the current state (Running, Updating, Error, ...)
	Phase   status.PhaseType     `json:"phase,omitempty"`
	Version status.VersionStatus `json:"version,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SetPhase sets the status phase on the DynaKube object
func (dk *EdgeConnectStatus) SetPhase(phase status.PhaseType) bool {
	upd := phase != dk.Phase
	dk.Phase = phase
	return upd
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EdgeConnect is the Schema for the EdgeConnect API
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=edgeconnects,scope=Namespaced,categories=dynatrace
// +kubebuilder:printcolumn:name="ApiServer",type=string,JSONPath=`.spec.apiServer`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:storageversion
type EdgeConnect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeConnectSpec   `json:"spec,omitempty"`
	Status EdgeConnectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EdgeConnectList contains a list of EdgeConnect
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
type EdgeConnectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeConnect `json:"items"`
}

func init() {
	v1alpha1.SchemeBuilder.Register(&EdgeConnect{}, &EdgeConnectList{})
}
