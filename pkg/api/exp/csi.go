package exp

import (
	"math"
	"time"
)

const (
	CSIMaxFailedMountAttemptsKey = FFPrefix + "max-csi-mount-attempts"
	CSIMaxMountTimeoutKey        = FFPrefix + "max-csi-mount-timeout"
	CSIReadOnlyVolumeKey         = FFPrefix + "injection-readonly-volume"
)

const (
	DefaultCSIMaxMountTimeout        = "10m"
	DefaultCSIMaxFailedMountAttempts = 10
)

func (ff *FeatureFlags) GetCSIMaxFailedMountAttempts() int {
	maxCsiMountAttemptsValue := ff.getFeatureFlagInt(CSIMaxFailedMountAttemptsKey, DefaultCSIMaxFailedMountAttempts)
	if maxCsiMountAttemptsValue < 0 {
		return DefaultCSIMaxFailedMountAttempts
	}

	return maxCsiMountAttemptsValue
}

func (ff *FeatureFlags) GetCSIMaxRetryTimeout() time.Duration {
	maxCsiMountTimeoutValue := ff.getFeatureFlagRaw(CSIMaxMountTimeoutKey)

	duration, err := time.ParseDuration(maxCsiMountTimeoutValue)
	if err != nil || duration < 0 {
		duration, _ = time.ParseDuration(DefaultCSIMaxMountTimeout)
	}

	return duration
}

// MountAttemptsToTimeout converts the (old) number of csi mount attempts into a time.Duration string.
// The converted value is based on the exponential backoff's algorithm.
// The output is string because it's main purpose is to convert the value of an annotation to another annotation.
func MountAttemptsToTimeout(maxAttempts int) string {
	var baseDelay = time.Second / 2

	delay := time.Duration(math.Exp2(float64(maxAttempts))) * baseDelay

	return delay.String()
}

func (ff *FeatureFlags) IsCSIVolumeReadOnly() bool {
	return ff.getFeatureFlagRaw(CSIReadOnlyVolumeKey) == truePhrase
}
