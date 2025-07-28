package activegate

import corev1 "k8s.io/api/core/v1"

type TemplateSpec struct {
	Image                         ImageRefSpec                      `json:"image,omitempty"`
	CustomPullSecret              string                            `json:"customPullSecret,omitempty"`
	Annotations                   map[string]string                 `json:"annotations,omitempty"`
	NodeSelector                  map[string]string                 `json:"nodeSelector,omitempty"`
	Replicas                      *int32                            `json:"replicas,omitempty"`
	Labels                        map[string]string                 `json:"labels,omitempty"`
	DNSPolicy                     corev1.DNSPolicy                  `json:"dnsPolicy,omitempty"`
	PriorityClassName             string                            `json:"priorityClassName,omitempty"`
	Resources                     corev1.ResourceRequirements       `json:"resources,omitempty"`
	Tolerations                   []corev1.Toleration               `json:"tolerations,omitempty"`
	Env                           []corev1.EnvVar                   `json:"env,omitempty"`
	TopologySpreadConstraints     []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	TerminationGracePeriodSeconds *int64                            `json:"terminationGracePeriodSeconds,omitempty"`
	PersistentStorage             PersistentStorageSpec             `json:"persistentStorage,omitempty"`
}

type ImageRefSpec struct {
	// Custom ActiveGate image repository
	Repository string `json:"repository,omitempty"`

	// Indicates version of the ActiveGate image to use
	Tag string `json:"tag,omitempty"`
}

type PersistentStorageSpec struct {
	Enabled             *bool                             `json:"enabled"`
	VolumeClaimTemplate *corev1.PersistentVolumeClaimSpec `json:"volumeClaimTemplate,omitempty"`
}
