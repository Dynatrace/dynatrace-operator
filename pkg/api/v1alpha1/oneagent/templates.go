package oneagent

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// If Image provided == enable
type HostAgentSpec struct {
	Image             ImageRefSpec                    `json:"image,omitempty"`
	CustomPullSecret  string                          `json:"customPullSecret,omitempty"`
	Annotations       map[string]string               `json:"annotations,omitempty"`
	NodeSelector      map[string]string               `json:"nodeSelector,omitempty"`
	Replicas          *int32                          `json:"replicas,omitempty"`
	Labels            map[string]string               `json:"labels,omitempty"`
	DNSPolicy         corev1.DNSPolicy                `json:"dnsPolicy,omitempty"`
	PriorityClassName string                          `json:"priorityClassName,omitempty"`
	UseLiveness       *bool                           `json:"liveness,omitempty"`
	Privileged        *bool                           `json:"privileged,omitempty"`
	SecCompProfile    string                          `json:"secCompProfile,omitempty"`
	Resources         corev1.ResourceRequirements     `json:"resources,omitempty"`
	Tolerations       []corev1.Toleration             `json:"tolerations,omitempty"`
	Args              []string                        `json:"args,omitempty"`
	Env               []corev1.EnvVar                 `json:"env,omitempty"`
	UpdateStrategy    *appsv1.DaemonSetUpdateStrategy `json:"updateStrategy,omitempty"`
	StorageHostPath   string                          `json:"storageHostPath,omitempty"` // TODO: maybe make nicer
}

type CodeModulesSpec struct {
	Image             ImageRefSpec                `json:"image"`
	CustomPullSecret  string                      `json:"customPullSecret,omitempty"`
	NamespaceSelector metav1.LabelSelector        `json:"namespaceSelector"`
	Resources         corev1.ResourceRequirements `json:"resources"`
	// TODO: add more as desired
}

type ImageRefSpec struct {
	// Custom ActiveGate image repository
	Repository string `json:"repository,omitempty"`

	// Indicates version of the ActiveGate image to use
	Tag string `json:"tag,omitempty"`
}
