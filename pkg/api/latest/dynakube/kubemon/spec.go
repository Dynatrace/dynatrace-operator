package kubemon

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:generate=true

// StatefulSetProperties holds standard Kubernetes scheduling/workload fields.
type StatefulSetProperties struct {
	// Amount of replicas for the KubernetesMonitoring pods.
	// Defaults to 1. Set >1 for HA mode (one active, others hot standby; backend elects leader).
	// +kubebuilder:validation:Optional
	Replicas *int32 `json:"replicas,omitempty"`

	// The KubernetesMonitoring container image. Defaults to the latest ActiveGate image provided by the registry on the tenant.
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// The KubernetesMonitoring container image pull policy.
	// +kubebuilder:validation:Optional
	ImagePullPolicy image.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Node selector to control the selection of nodes.
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Adds additional labels for the KubernetesMonitoring pods.
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Adds additional annotations to the KubernetesMonitoring pods.
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Define resources requests and limits for single KubernetesMonitoring pods.
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Define the rolling update strategy for the KubernetesMonitoring StatefulSet.
	// +kubebuilder:validation:Optional
	RollingUpdate *appsv1.RollingUpdateStatefulSetStrategy `json:"rollingUpdate,omitempty"`

	// Set tolerations for the KubernetesMonitoring pods.
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// List of environment variables to set for the KubernetesMonitoring pods.
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Adds TopologySpreadConstraints for the KubernetesMonitoring pods.
	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// Defines storage device.
	// +kubebuilder:validation:Optional
	VolumeClaimTemplate *corev1.PersistentVolumeClaimSpec `json:"volumeClaimTemplate,omitempty"`

	// Sets DNS Policy for the KubernetesMonitoring pods.
	// +kubebuilder:validation:Optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that name.
	// +kubebuilder:validation:Optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Configures the terminationGracePeriodSeconds parameter of the KubernetesMonitoring pod.
	// +kubebuilder:validation:Optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// Use an emptyDir volume instead of the PersistentVolumeClaim defined in volumeClaimTemplate.
	// Cached state is lost on pod restart; only enable for transient environments.
	// +kubebuilder:validation:Optional
	UseEphemeralVolume bool `json:"useEphemeralVolume,omitempty"`
}

// +kubebuilder:object:generate=true

type Spec struct {
	StatefulSetProperties `json:",inline"`

	// Add a custom properties file by providing it as a value or reference it from a secret.
	// +kubebuilder:validation:Optional
	CustomProperties *value.Source `json:"customProperties,omitempty"`

	// Set activation group for KubernetesMonitoring.
	// +kubebuilder:validation:Optional
	Group string `json:"group,omitempty"`

	// Reference to a secret containing the KubernetesMonitoring TLS cert+key and password.
	// +kubebuilder:validation:Optional
	TLSCertsRef *TLSCertsRef `json:"tlsCertsRef,omitempty"`

	// When present (even as {}), enables automatic cluster registration in Dynatrace.
	// +kubebuilder:validation:Optional
	Registration *Registration `json:"registration,omitempty"`
}

// TLSCertsRef references a Secret holding the KubernetesMonitoring TLS material.
// Expected Secret keys: server.p12 (certificate+key in pkcs12), password (passphrase for server.p12).
type TLSCertsRef struct {
	// Name of the Secret in the DynaKube namespace.
	// +kubebuilder:validation:Required
	SecretName string `json:"secretName"`
}

// Registration configures automatic cluster registration in Dynatrace.
// Presence (even as {}) enables registration — follows CRD presence-based enablement.
type Registration struct {
	// ClusterName is the display name used during registration. Defaults to the DynaKube name.
	// +kubebuilder:validation:Optional
	ClusterName string `json:"clusterName,omitempty"`

	// Enable the Kubernetes app (cluster details, workload views) in Dynatrace.
	// +kubebuilder:validation:Optional
	AppEnabled bool `json:"appEnabled,omitempty"`
}
