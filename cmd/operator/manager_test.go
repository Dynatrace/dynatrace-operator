package operator

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/stretchr/testify/assert"
)

func TestGetControllerAddFuncs(t *testing.T) {
	t.Run("without OLM", func(t *testing.T) {
		funcs := getControllerAddFuncs(false)

		assert.Len(t, funcs, 4) // dk, ec, nodes, certs
	})

	t.Run("with OLM", func(t *testing.T) {
		funcs := getControllerAddFuncs(true)

		assert.Len(t, funcs, 3) // dk, ec, nodes
	})

	t.Run("without HostAvailabilityDetectionEnvVar", func(t *testing.T) {
		t.Setenv(consts.HostAvailabilityDetectionEnvVar, "false")
		funcs := getControllerAddFuncs(true)

		assert.Len(t, funcs, 2) // dk, ec
	})
}

func TestShouldRunCRDStorageMigrationInInitManager(t *testing.T) {
	t.Run("should run if not set", func(t *testing.T) {
		os.Unsetenv("DT_CRD_STORAGE_MIGRATION")
		assert.True(t, shouldRunCRDStorageMigrationInitManager())
	})

	t.Run("should run if set to true", func(t *testing.T) {
		t.Setenv("DT_CRD_STORAGE_MIGRATION", "true")
		assert.True(t, shouldRunCRDStorageMigrationInitManager())
	})

	t.Run("should run if set to something random", func(t *testing.T) {
		t.Setenv("DT_CRD_STORAGE_MIGRATION", "job")
		assert.True(t, shouldRunCRDStorageMigrationInitManager())
	})

	t.Run("should not run if set to false", func(t *testing.T) {
		t.Setenv("DT_CRD_STORAGE_MIGRATION", "false")
		assert.False(t, shouldRunCRDStorageMigrationInitManager())
	})
}
