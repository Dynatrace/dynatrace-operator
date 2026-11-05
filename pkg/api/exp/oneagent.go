// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package exp

import (
	"strconv"
	"strings"
)

const (
	// Deprecated: OAProxyIgnoredKey use NoProxy annotation instead.
	OAProxyIgnoredKey = FFPrefix + "oneagent-ignore-proxy"

	// Deprecated: OAMaxUnavailableKey is no longer used. Use rollingUpdate configuration for OneAgent in the DynaKube spec instead.
	OAMaxUnavailableKey      = FFPrefix + "oneagent-max-unavailable"
	OAInitialConnectRetryKey = FFPrefix + "oneagent-initial-connect-retry-ms"
	OAPrivilegedKey          = FFPrefix + "oneagent-privileged"
	OASkipLivenessProbeKey   = FFPrefix + "oneagent-skip-liveness-probe"
	OAClassicNonRootKey      = FFPrefix + "oneagent-classic-nonroot"
	OAImageVolumeKey         = FFPrefix + "mount-code-modules-via-image-volume"
	OANodeImagePullKey       = FFPrefix + "node-image-pull"
	// OANodeImagePullTechnologiesKey can be set on a Pod or DynaKube to configure which code module technologies to download. It's set to
	// "all" if not set.
	OANodeImagePullTechnologiesKey = "oneagent.dynatrace.com/technologies"
)

const (
	DefaultOAIstioInitialConnectRetry = 6000

	// DefaultOAClassicNonRootID is the user and group ID used for classic OneAgent when no explicit IDs are configured.
	DefaultOAClassicNonRootID int64 = 1000
)

// Deprecated: Use rollingUpdate configuration for OneAgent in the DynaKube spec instead.
// GetOneAgentMaxUnavailable is a feature flag to configure maxUnavailable on the OneAgent DaemonSets rolling upgrades.
func (ff *FeatureFlags) GetOneAgentMaxUnavailable() int {
	return ff.getIntWithDefault(OAMaxUnavailableKey, 1)
}

// Deprecated: Use NoProxy annotation instead.
// OneAgentIgnoresProxy is a feature flag to ignore the proxy for oneAgents when set in CR.
func (ff *FeatureFlags) OneAgentIgnoresProxy() bool {
	return ff.getBoolWithDefault(OAProxyIgnoredKey, false)
}

// GetAgentInitialConnectRetry is a feature flag to configure startup delay of standalone agents.
func (ff *FeatureFlags) GetAgentInitialConnectRetry(isIstio bool) int {
	defaultValue := -1
	ffValue := ff.getIntWithDefault(OAInitialConnectRetryKey, defaultValue)

	// In case of istio, we want to have a longer initial delay for codemodules to ensure the DT service is created consistently
	if ffValue == defaultValue && isIstio {
		ffValue = DefaultOAIstioInitialConnectRetry
	}

	return ffValue
}

func (ff *FeatureFlags) IsOneAgentPrivileged() bool {
	return ff.getBoolWithDefault(OAPrivilegedKey, false)
}

// GetClassicOneAgentNonRoot returns the user and group the classic OneAgent should run as, and whether non-root is enabled at all.
//
// The feature flag is disabled when not set. When set, the value is "<uid>:<gid>";
// an empty value or invalid value selects DefaultOAClassicNonRootID for both.
func (ff *FeatureFlags) GetClassicOneAgentNonRoot() (user, group int64, enabled bool) {
	if !ff.IsSet(OAClassicNonRootKey) {
		return 0, 0, false
	}

	raw := strings.TrimSpace(ff.getRaw(OAClassicNonRootKey))
	user, group = DefaultOAClassicNonRootID, DefaultOAClassicNonRootID

	userRaw, groupRaw, found := strings.Cut(raw, ":")
	if !found {
		return user, group, true
	}

	parsedUser, userErr := strconv.ParseInt(strings.TrimSpace(userRaw), 10, 64)
	parsedGroup, groupErr := strconv.ParseInt(strings.TrimSpace(groupRaw), 10, 64)

	if userErr != nil || groupErr != nil || parsedUser < 0 || parsedGroup < 0 {
		return user, group, true
	}

	return parsedUser, parsedGroup, true
}

func (ff *FeatureFlags) SkipOneAgentLivenessProbe() bool {
	return ff.getBoolWithDefault(OASkipLivenessProbeKey, false)
}

func (ff *FeatureFlags) IsCodeModuleImageVolume() bool {
	return ff.getBoolWithDefault(OAImageVolumeKey, false)
}

func (ff *FeatureFlags) IsNodeImagePull() bool {
	return ff.getBoolWithDefault(OANodeImagePullKey, false)
}

func (ff *FeatureFlags) GetNodeImagePullTechnology() string {
	return ff.getRaw(OANodeImagePullTechnologiesKey)
}
