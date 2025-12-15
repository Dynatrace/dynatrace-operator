// The CodeModule installer Job configuration settings is meant to be partly inherited from the CSI DaemonSet settings and partly
// defined specifically for the Job. These settings are defined in the Helm chart and passed to the CSI driver via an environment
// variable in JSON format. This file contains the logic to read and parse these settings.
package helmconfig

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
	JSONEnv = "helm.json"
)

var (
	once sync.Once

	conf Config

	fallback = Config{
		CSIDaemonSetConfig: CSIDaemonSetConfig{
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
		},
		Job: JobConfig{
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
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("30Mi"),
				},
			},
		},
	}

	log = logd.Get().WithName("csi-job")
)

// JobConfig holds settings specific to the CodeModule installer Job that were defined in the helm chart.
type JobConfig struct {
	SecurityContext corev1.SecurityContext      `json:"securityContext"`
	Resources       corev1.ResourceRequirements `json:"resources"`
}

// CSIDaemonSetConfig holds settings inherited from the CSI DaemonSet.
type CSIDaemonSetConfig struct {
	Tolerations []corev1.Toleration `json:"tolerations"`
	Annotations map[string]string   `json:"annotations"`
	Labels      map[string]string   `json:"labels"`
}

// Config holds all settings relevant for the CodeModule installer Job that were defined in the helm chart.
type Config struct {
	CSIDaemonSetConfig `json:",inline"`
	Job                JobConfig `json:"job"`
}

func Get() Config {
	Read()

	return conf
}

func Read() {
	once.Do(func() {
		confJSON := os.Getenv(JSONEnv)
		if confJSON == "" {
			log.Info("envvar not set, using default", "envvar", JSONEnv)

			conf = fallback

			return
		}

		err := json.Unmarshal([]byte(confJSON), &conf)
		if err != nil {
			log.Info("problem unmarshalling envvar content, using default", "envvar", JSONEnv, "err", err)

			conf = fallback

			return
		}

		log.Info("envvar content read and set", "envvar", JSONEnv, "value", confJSON)
	})
}
