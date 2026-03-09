package envvars

import (
	"strconv"
	"testing"
	"time"

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

func TestGetDurationMinutes(t *testing.T) {
	const testEnvVar = "TEST_DURATION_MIN"
	const defaultDuration = 30 * time.Minute

	t.Run("env var is set to valid minutes and properly parsed", func(t *testing.T) {
		t.Setenv(testEnvVar, "15")

		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, 15*time.Minute, duration)
	})

	t.Run("env var is set to zero and returns default", func(t *testing.T) {
		t.Setenv(testEnvVar, "0")

		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, defaultDuration, duration)
	})

	t.Run("env var is set to large value and properly parsed", func(t *testing.T) {
		t.Setenv(testEnvVar, "1440")

		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, 1440*time.Minute, duration)
		assert.Equal(t, 24*time.Hour, duration)
	})

	t.Run("env var is set to invalid value and returns default", func(t *testing.T) {
		t.Setenv(testEnvVar, "invalid")

		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, defaultDuration, duration)
	})

	t.Run("env var is set to negative value and returns default", func(t *testing.T) {
		t.Setenv(testEnvVar, "-10")

		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, defaultDuration, duration)
	})

	t.Run("env var is set to float value and returns default", func(t *testing.T) {
		t.Setenv(testEnvVar, "15.5")

		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, defaultDuration, duration)
	})

	t.Run("env var is NOT set and returns default", func(t *testing.T) {
		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, defaultDuration, duration)
	})

	t.Run("env var is empty string and returns default", func(t *testing.T) {
		t.Setenv(testEnvVar, "")

		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, defaultDuration, duration)
	})

	t.Run("different default values are respected", func(t *testing.T) {
		testCases := []time.Duration{
			5 * time.Minute,
			15 * time.Minute,
			60 * time.Minute,
			2 * time.Hour,
		}

		for _, defaultVal := range testCases {
			duration := GetDurationMinutes("NONEXISTENT_VAR", defaultVal)

			assert.Equal(t, defaultVal, duration)
		}
	})

	t.Run("env var is set above maximum and is capped", func(t *testing.T) {
		t.Setenv(testEnvVar, strconv.FormatInt(maxDurationMinutes+1, 10))

		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, time.Duration(maxDurationMinutes)*time.Minute, duration)
	})

	t.Run("env var is set to maximum and properly parsed", func(t *testing.T) {
		t.Setenv(testEnvVar, strconv.FormatInt(maxDurationMinutes, 10))

		duration := GetDurationMinutes(testEnvVar, defaultDuration)

		assert.Equal(t, time.Duration(maxDurationMinutes)*time.Minute, duration)
	})
}
