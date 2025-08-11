package extension

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
)

type Extensions struct {
	ExecutionController    *ExecutionControllerSpec
	OpenTelemetryCollector *OpenTelemetryCollectorSpec

	name      string
	namespace string
	enabled   bool
}

func (e *Extensions) SetName(name string) {
	e.name = name
}

func (e *Extensions) SetNamespace(namespace string) {
	e.namespace = namespace
}

func (e *Extensions) SetEnabled(enabled bool) {
	e.enabled = enabled
}

func (e *Extensions) Enabled() bool {
	return e.enabled
}

func (e *Extensions) TLSRefName() string {
	return e.ExecutionController.TLSRefName
}

func (e *Extensions) NeedsSelfSignedTLS() bool {
	return e.TLSRefName() == ""
}

func (e *Extensions) TLSSecretName() string {
	if e.NeedsSelfSignedTLS() {
		return e.SelfSignedTLSSecretName()
	}

	return e.TLSRefName()
}

func (e *Extensions) SelfSignedTLSSecretName() string {
	return e.name + consts.ExtensionsSelfSignedTLSSecretSuffix
}

func (e *Extensions) ExecutionControllerStatefulsetName() string {
	return e.name + "-extensions-controller"
}

func (e *Extensions) TokenSecretName() string {
	return e.name + "-extensions-token"
}

func (e *Extensions) PortName() string {
	return "dynatrace" + consts.ExtensionsControllerSuffix + "-" + consts.ExtensionsCollectorTargetPortName
}

func (e *Extensions) ServiceNameFQDN() string {
	return e.ServiceName() + "." + e.namespace
}

func (e *Extensions) ServiceName() string {
	return e.name + consts.ExtensionsControllerSuffix
}

// +kubebuilder:object:generate=true

type Spec struct {
}

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

type OpenTelemetryCollectorSpec struct {

	// Adds additional labels for the OtelCollector pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Adds additional annotations to the OtelCollector pods
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Number of replicas for your OtelCollector
	// +kubebuilder:validation:Optional
	Replicas *int32 `json:"replicas"`

	// Overrides the default image
	// +kubebuilder:validation:Optional
	ImageRef image.Ref `json:"imageRef"`

	// +kubebuilder:validation:Optional
	TLSRefName string `json:"tlsRefName,omitempty"`

	// Define resources' requests and limits for single OtelCollector pod
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources"`

	// Set tolerations for the OtelCollector pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Adds TopologySpreadConstraints for the OtelCollector pods
	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}
