package helmconfig

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestGet(t *testing.T) {
	t.Run("empty env -> use fallback", func(t *testing.T) {
		t.Setenv(JSONEnv, "")

		m := Get()
		assert.Equal(t, fallback, m)

		once = sync.Once{} // need to reset it
		conf = Config{}
	})

	t.Run("messy env -> use fallback", func(t *testing.T) {
		t.Setenv(JSONEnv, "this is not json :(")

		m := Get()
		assert.Equal(t, fallback, m)

		once = sync.Once{} // need to reset it
		conf = Config{}
	})

	t.Run("correct env -> set correctly", func(t *testing.T) {
		jsonValue := `
		{
			"tolerations": [{"effect":"NoSchedule","key":"test-key1","operator":"Exists"},{"effect":"NoSchedule","key":"test-key2","operator":"Exists"}],
			"annotations": {"test-annotation1":"test-value1","test-annotation2":"test-value2"},
			"labels": {"test-label1":"test-value1","test-label2":"test-value2"},
			"job": {
			  "securityContext": {"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL","DAC_OVERRIDE"]},"privileged":false,"readOnlyRootFilesystem":false,"runAsNonRoot":false,"runAsUser":0,"seLinuxOptions":{"level":"s1"},"seccompProfile":{"type":"RuntimeDefault"}},
              "resources": {"requests":{"cpu":"100m","memory":"200Mi"},"limits":{"cpu":"500m","memory":"500Mi"}}
			}
		}`
		expected := Config{
			CSIDaemonSetConfig: CSIDaemonSetConfig{
				Tolerations: []corev1.Toleration{
					{
						Effect:   corev1.TaintEffectNoSchedule,
						Key:      "test-key1",
						Operator: corev1.TolerationOpExists,
					},
					{
						Effect:   corev1.TaintEffectNoSchedule,
						Key:      "test-key2",
						Operator: corev1.TolerationOpExists,
					},
				},
				Annotations: map[string]string{
					"test-annotation1": "test-value1",
					"test-annotation2": "test-value2",
				},
				Labels: map[string]string{
					"test-label1": "test-value1",
					"test-label2": "test-value2",
				},
			},
			Job: JobConfig{
				SecurityContext: corev1.SecurityContext{
					AllowPrivilegeEscalation: ptr.To(false),
					Privileged:               ptr.To(false),
					ReadOnlyRootFilesystem:   ptr.To(false),
					RunAsNonRoot:             ptr.To(false),
					RunAsUser:                ptr.To(int64(0)),
					SELinuxOptions: &corev1.SELinuxOptions{
						Level: "s1",
					},
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{
							"ALL",
							"DAC_OVERRIDE",
						},
					},
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("500Mi"),
					},
				},
			},
		}

		t.Setenv(JSONEnv, jsonValue)

		m := Get()
		assert.Equal(t, expected, m)

		once = sync.Once{} // need to reset it
		conf = Config{}
	})

	t.Run("run only once", func(t *testing.T) {
		jsonValue := `
		{
			"job": {
              "securityContext": {"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"privileged":false,"readOnlyRootFilesystem":true,"runAsNonRoot":false,"runAsUser":0,"seLinuxOptions":{"level":"s0"},"seccompProfile":{"type":"RuntimeDefault"}},
              "resources": {"requests":{"cpu":"300m","memory":"100Mi"}},
			}
		}`

		t.Setenv(JSONEnv, jsonValue)

		m := Get()
		assert.Equal(t, fallback, m)

		t.Setenv(JSONEnv, "boom")

		m = Get()
		assert.Equal(t, fallback, m)

		once = sync.Once{} // need to reset it
		conf = Config{}
	})
}
