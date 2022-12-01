package kubeobjects

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolvePlatformFromEnv(t *testing.T) {
	t.Run("openshift", func(t *testing.T) {
		os.Setenv(platformEnvName, openshiftPlatformEnvValue)
		assert.Equal(t, Openshift, ResolvePlatformFromEnv())
	})
	t.Run("kubernetes explicitly", func(t *testing.T) {
		os.Setenv(platformEnvName, kubernetesPlatformEnvValue)
		assert.Equal(t, Kubernetes, ResolvePlatformFromEnv())
	})
	t.Run("kubernetes default", func(t *testing.T) {
		os.Setenv(platformEnvName, "asd")
		assert.Equal(t, Kubernetes, ResolvePlatformFromEnv())
		os.Setenv(platformEnvName, "")
		assert.Equal(t, Kubernetes, ResolvePlatformFromEnv())
	})
}
