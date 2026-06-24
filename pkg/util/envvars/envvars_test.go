package envvars

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/stretchr/testify/assert"
)

func TestGetDuration(t *testing.T) {
	const testVar = "TEST_DURATION_VAR"

	t.Run("env var not set returns default", func(t *testing.T) {
		assert.Equal(t, 5*time.Second, GetDuration(t.Context(), testVar, 5*time.Second))
	})

	t.Run("env var set to valid duration returns parsed value", func(t *testing.T) {
		t.Setenv(testVar, "10m")
		assert.Equal(t, 10*time.Minute, GetDuration(t.Context(), testVar, 5*time.Second))
	})

	t.Run("env var set to invalid string returns default", func(t *testing.T) {
		t.Setenv(testVar, "not-a-duration")
		assert.Equal(t, 5*time.Second, GetDuration(t.Context(), testVar, 5*time.Second))
	})
}

func TestGetBool(t *testing.T) {
	t.Run("env var is set to true and properly parsed to true", func(t *testing.T) {
		t.Setenv(consts.HostAvailabilityDetectionEnvVar, "true")

		assert.True(t, GetBool(consts.HostAvailabilityDetectionEnvVar, true))
	})

	t.Run("env var is set to false and properly parsed to false", func(t *testing.T) {
		t.Setenv(consts.HostAvailabilityDetectionEnvVar, "false")

		assert.False(t, GetBool(consts.HostAvailabilityDetectionEnvVar, true))
	})

	t.Run("env var is set to dummy and properly parsed to default value", func(t *testing.T) {
		t.Setenv(consts.HostAvailabilityDetectionEnvVar, "dummy")

		assert.True(t, GetBool(consts.HostAvailabilityDetectionEnvVar, true))
	})

	t.Run("env var is NOT set and fallback to default", func(t *testing.T) {
		assert.True(t, GetBool(consts.HostAvailabilityDetectionEnvVar, true))
		assert.False(t, GetBool(consts.HostAvailabilityDetectionEnvVar, false))
	})
}
