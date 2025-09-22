package envvars

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/stretchr/testify/assert"
)

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
