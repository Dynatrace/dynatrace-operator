// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1alpha1
package edgeconnect

import (
	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	"github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EdgeConnectSpec defines the desired state of EdgeConnect
type EdgeConnectSpec struct {
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Location of the Dynatrace API to connect to, including your specific environment UUID
	// +required
	// +kubebuilder:validation:Required
	ApiServer string `json:"apiServer"`

	// EdgeConnect uses the OAuth client to authenticate itself with the Dynatrace platform.
	// +required
	// +kubebuilder:validation:Required
	Oauth OAuthSpec `json:"oauth"`

	// Optional: restrict outgoing HTTP requests to your internal resources to specified hosts
	// +kubebuilder:validation:Optional
	HostRestrictions string `json:"hostRestrictions,omitempty"`

	// Optional: image reference
	// +kubebuilder:validation:Optional
	ImageRef ImageRefSpec `json:"imageRef,omitempty"`

	// Optional: enables automatic restarts of EdgeConnect pods in case a new version is available (the default value is: true)
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	AutoUpdate bool `json:"autoUpdate"`

	// Optional: pull secret for your private registry
	// +kubebuilder:validation:Optional
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// Optional: adds additional annotations for the EdgeConnect pods
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Optional: adds additional labels for the EdgeConnect pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Optional: adds additional environment variables for the EdgeConnect pods
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Amount of replicas for your EdgeConnect (the default value is: 1)
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=1
	Replicas *int32 `json:"replicas"`

	// Optional: define resources requests and limits for single pods
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: node selector to control the selection of nodes for the EdgeConnect pods
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
	// The secret from your Oauth client generation
	// +required
	// +kubebuilder:validation:Required
	ClientSecret string `json:"clientSecret"`
	// The token endpoint URL
	// +required
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`
	// URN identifying your account. You get the URN when creating the OAuth client
	// +required
	// +kubebuilder:validation:Required
	Resource string `json:"resource"`
}

type ImageRefSpec struct {
	// Optional: if specified, indicates the EdgeConnect repository to use
	// +kubebuilder:validation:Optional
	Repository string `json:"repository,omitempty"`

	// Optional: indicates version of the EdgeConnect image to use
	// +kubebuilder:validation:Optional
	Tag string `json:"tag,omitempty"`
}

// EdgeConnectStatus defines the observed state of EdgeConnect
type EdgeConnectStatus struct {
	// Defines the current state (Running, Updating, Error, ...)
	Phase status.PhaseType `json:"phase,omitempty"`

	// State of the current image
	Version status.VersionStatus `json:"version,omitempty"`

	// Indicates when the instance was last updated
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SetPhase sets the status phase on the EdgeConnect object
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
