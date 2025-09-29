package extensions

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	corev1 "k8s.io/api/core/v1"
)

type Extensions struct {
	*ExecutionControllerSpec

	name      string
	namespace string
	enabled   bool
}

// +kubebuilder:object:generate=true
type Spec struct {
	PrometheusSpec *PrometheusSpec `json:"prometheus,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxItems:=1
	// +kubebuilder:validation:MinItems:=1
	DatabasesSpec []DatabaseSpec `json:"databases,omitempty"`
}

// +kubebuilder:object:generate=true
type PrometheusSpec struct{}

// +kubebuilder:object:generate=true
type ExecutionControllerSpec struct {

	// Defines storage device
	// +kubebuilder:validation:Optional
	PersistentVolumeClaim *corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaim,omitempty"`

	// Adds additional labels for the ExtensionExecutionController pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Adds additional annotations to the ExtensionExecutionController pods
	Annotations map[string]string `json:"annotations,omitempty"`

	// Overrides the default image
	// +kubebuilder:validation:Optional
	ImageRef image.Ref `json:"imageRef"`

	// +kubebuilder:validation:Optional
	TLSRefName string `json:"tlsRefName,omitempty"`

	// Defines name of ConfigMap containing custom configuration file
	// +kubebuilder:validation:Optional
	CustomConfig string `json:"customConfig,omitempty"`

	// Defines name of Secret containing certificates for custom extensions signature validation
	// +kubebuilder:validation:Optional
	CustomExtensionCertificates string `json:"customExtensionCertificates,omitempty"`

	// Define resources' requests and limits for single ExtensionExecutionController pod
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources"`

	// Set tolerations for the ExtensionExecutionController pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Adds TopologySpreadConstraints for the ExtensionExecutionController pods
	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	// Selects EmptyDir volume to be storage device
	// +kubebuilder:validation:Optional
	UseEphemeralVolume bool `json:"useEphemeralVolume,omitempty"`
}

// +kubebuilder:object:generate=true
type DatabaseSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=8
	// +kubebuilder:validation:Pattern:=`^[a-z0-9-]+$`
	ID string `json:"id"`

	// +kubebuilder:validation:Optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// +kubebuilder:validation:Optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// +kubebuilder:validation:Optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +kubebuilder:validation:Optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// +kubebuilder:validation:Optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// +kubebuilder:object:generate=true
type DatabaseExecutorSpec struct {
	// +kubebuilder:validation:Required
	ImageRef image.Ref `json:"imageRef"`

	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}
