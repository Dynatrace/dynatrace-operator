// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package exp

import (
	"time"
)

const (
	CSIMaxMountTimeoutKey = FFPrefix + "max-csi-mount-timeout"
)

const (
	DefaultCSIMaxMountTimeout = "10m"
)

func (ff *FeatureFlags) GetCSIMaxRetryTimeout() time.Duration {
	maxCSIMountTimeoutValue := ff.getRaw(CSIMaxMountTimeoutKey)

	duration, err := time.ParseDuration(maxCSIMountTimeoutValue)
	if err != nil || duration < 0 {
		duration, _ = time.ParseDuration(DefaultCSIMaxMountTimeout)
	}

	return duration
}
