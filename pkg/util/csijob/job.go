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
	SettingsJsonEnv = "job.json"
)

var (
	once sync.Once

	settings Settings

	fallbackSettings = Settings{
		SecurityContext: corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Privileged:               ptr.To(false),
			ReadOnlyRootFilesystem:   ptr.To(true),
			RunAsNonRoot:             ptr.To(false),
			RunAsUser:                ptr.To(int64(0)),
			SELinuxOptions: &corev1.SELinuxOptions{
				Level: "s0",
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("300m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
		},
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
	}

	log = logd.Get().WithName("csi-job")
)

type Settings struct {
	SecurityContext corev1.SecurityContext      `json:"securityContext"`
	Annotations     map[string]string           `json:"annotations"`
	Labels          map[string]string           `json:"labels"`
	Resources       corev1.ResourceRequirements `json:"resources"`
	Tolerations     []corev1.Toleration         `json:"tolerations"`
}

func GetSettings() Settings {
	ReadSettings()

	return settings
}

func ReadSettings() {
	once.Do(func() {
		settingsJson := os.Getenv(SettingsJsonEnv)
		if settingsJson == "" {
			log.Info("envvar not set, using default", "envvar", SettingsJsonEnv)

			settings = fallbackSettings

			return
		}

		err := json.Unmarshal([]byte(settingsJson), &settings)
		if err != nil {
			log.Info("problem unmarshalling envvar content, using default", "envvar", SettingsJsonEnv, "err", err)

			settings = fallbackSettings

			return
		}

		log.Info("envvar content read and set", "envvar", SettingsJsonEnv, "value", settingsJson)
	})
}
