package version

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/stretchr/testify/assert"
)

func TestUserAgent(t *testing.T) {
	restoreVersion := Version
	t.Cleanup(func() { Version = restoreVersion })

	t.Run("default", func(t *testing.T) {
		Version = "snapshot-test"
		assert.Equal(t, "dynatrace-operator/snapshot-test", UserAgent())
	})

	t.Run("default", func(t *testing.T) {
		Version = "snapshot-test-" + arch.ImageArch
		assert.Equal(t, "dynatrace-operator/snapshot-test", UserAgent())
	})
}
