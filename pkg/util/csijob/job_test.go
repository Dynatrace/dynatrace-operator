package csijob

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
		t.Setenv(SettingsJsonEnv, "")

		m := GetSettings()
		assert.Equal(t, fallbackSettings, m)

		once = sync.Once{} // need to reset it
		settings = Settings{}
	})

	t.Run("messy env -> use fallback", func(t *testing.T) {
		t.Setenv(SettingsJsonEnv, "this is not json :(")

		m := GetSettings()
		assert.Equal(t, fallbackSettings, m)

		once = sync.Once{} // need to reset it
		settings = Settings{}
	})

	t.Run("correct env -> set correctly", func(t *testing.T) {
		jsonValue := `
		{
            "securityContext": {"allowPrivilegeEscalation":true,"capabilities":{"drop":["ALL","DAC_OVERRIDE"]},"privileged":true,"readOnlyRootFilesystem":false,"runAsNonRoot":false,"runAsUser":0,"seLinuxOptions":{"level":"s1"},"seccompProfile":{"type":"RuntimeDefault"}},
            "resources": {"requests":{"cpu":"300m","memory":"100Mi"},"limits":{"cpu":"500m","memory":"500Mi"}}
		}`
		expected := Settings{
			SecurityContext: corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.To(true),
				Privileged:               ptr.To(true),
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
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
		}

		t.Setenv(SettingsJsonEnv, jsonValue)

		m := GetSettings()
		assert.Equal(t, expected, m)

		once = sync.Once{} // need to reset it
		settings = Settings{}
	})

	t.Run("run only once", func(t *testing.T) {
		jsonValue := `
		{
            "securityContext": {"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"privileged":false,"readOnlyRootFilesystem":true,"runAsNonRoot":false,"runAsUser":0,"seLinuxOptions":{"level":"s0"},"seccompProfile":{"type":"RuntimeDefault"}},
            "resources": {"requests":{"cpu":"300m","memory":"100Mi"}},
		}`

		t.Setenv(SettingsJsonEnv, jsonValue)

		m := GetSettings()
		assert.Equal(t, fallbackSettings, m)

		t.Setenv(SettingsJsonEnv, "boom")

		m = GetSettings()
		assert.Equal(t, fallbackSettings, m)

		once = sync.Once{} // need to reset it
		settings = Settings{}
	})
}
