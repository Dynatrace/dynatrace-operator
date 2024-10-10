/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package exp

import (
	"math"
	"strconv"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	AnnotationPrefix = "feature.dynatrace.com/"

	// General.
	PublicRegistryAnnotation = AnnotationPrefix + "public-registry"
	NoProxyAnnotation        = AnnotationPrefix + "no-proxy"

	// CSI.
	MaxFailedCsiMountAttemptsAnnotation = AnnotationPrefix + "max-csi-mount-attempts"
	MaxCsiMountTimeoutAnnotation        = AnnotationPrefix + "max-csi-mount-timeout"
	ReadOnlyCsiVolumeAnnotation         = AnnotationPrefix + "injection-readonly-volume"

	falsePhrase  = "false"
	truePhrase   = "true"
	silentPhrase = "silent"
	failPhrase   = "fail"
)

const (
	DefaultMaxCsiMountTimeout         = "10m"
	DefaultMaxFailedCsiMountAttempts  = 10
	DefaultMinRequestThresholdMinutes = 15
)

type FeatureFlags struct {
	annotations map[string]string
}

func NewFeatureFlags(annotations map[string]string) *FeatureFlags {
	if annotations == nil {
		annotations = map[string]string{}
	}

	return &FeatureFlags{
		annotations: annotations,
	}
}

var (
	log = logd.Get().WithName("feature-flags")
)

func (f *FeatureFlags) getDisableFlagWithDeprecatedAnnotation(key string, deprecatedKey string) bool {
	return f.getRawValue(key) == falsePhrase ||
		f.getRawValue(deprecatedKey) == truePhrase && f.getRawValue(key) == ""
}

func (f *FeatureFlags) getRawValue(key string) string {
	if raw, ok := f.annotations[key]; ok {
		return raw
	}

	return ""
}

func (f *FeatureFlags) getIntValue(key string, defaultVal int) int {
	raw := f.getRawValue(key)
	if raw == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}

	return val
}

// FeatureNoProxy is a feature flag to set the NO_PROXY value to be used by the dtClient.
func (f *FeatureFlags) GetNoProxy() string {
	return f.getRawValue(NoProxyAnnotation)
}

func (f *FeatureFlags) GetMaxFailedCsiMountAttempts() int {
	maxCsiMountAttemptsValue := f.getIntValue(MaxFailedCsiMountAttemptsAnnotation, DefaultMaxFailedCsiMountAttempts)
	if maxCsiMountAttemptsValue < 0 {
		return DefaultMaxFailedCsiMountAttempts
	}

	return maxCsiMountAttemptsValue
}

func (f *FeatureFlags) GetMaxCSIRetryTimeout() time.Duration {
	maxCsiMountTimeoutValue := f.getRawValue(MaxCsiMountTimeoutAnnotation)

	duration, err := time.ParseDuration(maxCsiMountTimeoutValue)
	if err != nil || duration < 0 {
		duration, _ = time.ParseDuration(DefaultMaxCsiMountTimeout)
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

func (f *FeatureFlags) ISCsiVolumeReadOnly() bool {
	return f.getRawValue(ReadOnlyCsiVolumeAnnotation) == truePhrase
}

func (f *FeatureFlags) IsPublicRegistryEnabled() bool {
	return f.getRawValue(PublicRegistryAnnotation) == truePhrase
}
