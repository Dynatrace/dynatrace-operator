package version

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestIsRemoteClusterVersionSupported(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout))

	t.Run("IsRemoteClusterVersionSupported", func(t *testing.T) {
		isSupported := IsRemoteClusterVersionSupported(logger, "1.205.0")
		assert.True(t, isSupported)
	})
	t.Run("IsRemoteClusterVersionSupported unsupported version", func(t *testing.T) {
		isSupported := IsRemoteClusterVersionSupported(logger, "0.000.0")
		assert.False(t, isSupported)
	})
	t.Run("IsRemoteClusterVersionSupported dtclient is nil", func(t *testing.T) {
		isSupported := IsRemoteClusterVersionSupported(logger, "")
		assert.False(t, isSupported)
	})
}

func TestIsSupportedClusterVersion(t *testing.T) {
	t.Run("IsSupportedClusterVersion", func(t *testing.T) {
		a := versionInfo{
			major:   2,
			minor:   0,
			release: 0,
		}
		isSupported := isSupportedClusterVersion(a)
		assert.True(t, isSupported)

		a = minSupportedClusterVersion
		isSupported = isSupportedClusterVersion(a)
		assert.True(t, isSupported)

		a = versionInfo{
			major:   1,
			minor:   196,
			release: 10000,
		}
		isSupported = isSupportedClusterVersion(a)
		assert.False(t, isSupported)
	})
}
