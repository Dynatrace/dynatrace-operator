package csijob

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

const (
	SettingsJSONEnv = "helm.json"
)

var (
	once sync.Once

	settings Settings

	fallbackSettings = Settings{
		Tolerations: []corev1.Toleration{
			{
				Effect:   corev1.TaintEffectNoSchedule,
				Key:      "node-role.kubernetes.io/master",
				Operator: corev1.TolerationOpExists,
			},
			{
				Effect:   corev1.TaintEffectNoSchedule,
				Key:      "node-role.kubernetes.io/control-plane",
				Operator: corev1.TolerationOpExists,
			},
		},
		Job: JobSettings{
			SecurityContext: corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.To(true),
				Privileged:               ptr.To(true),
				ReadOnlyRootFilesystem:   ptr.To(true),
				RunAsNonRoot:             ptr.To(false),
				RunAsUser:                ptr.To(int64(0)),
				SELinuxOptions: &corev1.SELinuxOptions{
					Level: "s0",
				},
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
			PriorityClassName: "dynatrace-high-priority",
		},
	}

	log = logd.Get().WithName("csi-job")
)

type JobSettings struct {
	SecurityContext   corev1.SecurityContext      `json:"securityContext"`
	Resources         corev1.ResourceRequirements `json:"resources"`
	PriorityClassName string                      `json:"priorityClassName"`
}
type Settings struct {
	Annotations map[string]string   `json:"annotations"`
	Labels      map[string]string   `json:"labels"`
	Tolerations []corev1.Toleration `json:"tolerations"`
	Job         JobSettings         `json:"job"`
}

func GetSettings() Settings {
	ReadSettings()

	return settings
}

func ReadSettings() {
	once.Do(func() {
		settingsJSON := os.Getenv(SettingsJSONEnv)
		if settingsJSON == "" {
			log.Info("envvar not set, using default", "envvar", SettingsJSONEnv)

			settings = fallbackSettings

			return
		}

		err := json.Unmarshal([]byte(settingsJSON), &settings)
		if err != nil {
			log.Info("problem unmarshalling envvar content, using default", "envvar", SettingsJSONEnv, "err", err)

			settings = fallbackSettings

			return
		}

		log.Info("envvar content read and set", "envvar", SettingsJSONEnv, "value", settingsJSON)
	})
}
