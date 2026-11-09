// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package exp

import (
	"context"
	"errors"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8ssecuritycontext"
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
// The feature flag is disabled when not set. When set, the value is either empty, which selects
// DefaultOAClassicNonRootID for both user and group, or "<uid>:<gid>" with a non-root (non-zero) user ID.
// A value in any other format, or one resulting in a root user, is invalid; it is logged and the flag is ignored (enabled == false).
func (ff *FeatureFlags) GetClassicOneAgentNonRoot(ctx context.Context) (user, group int64, enabled bool) {
	_, log := logd.NewFromContext(ctx, "exp")

	if !ff.IsSet(OAClassicNonRootKey) {
		return 0, 0, false
	}

	raw := ff.getRaw(OAClassicNonRootKey)
	if len(raw) == 0 {
		log.Debug("oneagent-nonroot FF present and empty, using default nonroot user/group.", "defaultID", DefaultOAClassicNonRootID)

		return DefaultOAClassicNonRootID, DefaultOAClassicNonRootID, true
	}

	userRaw, groupRaw, found := strings.Cut(strings.TrimSpace(raw), ":")
	if !found {
		log.Info("oneagent-nonroot FF present, but has unexpected format, ignoring flag, expected format \"\" or \"userID:groupID\"", "value", raw)

		return 0, 0, false
	}

	parsedUser, userErr := k8ssecuritycontext.GetValidatedRunAsID(strings.TrimSpace(userRaw))
	parsedGroup, groupErr := k8ssecuritycontext.GetValidatedRunAsID(strings.TrimSpace(groupRaw))

	if err := errors.Join(userErr, groupErr); err != nil {
		log.Error(err, "oneagent-nonroot FF present, but failed to parse, ignoring flag", "userRaw", userRaw, "groupRaw", groupRaw)

		return 0, 0, false
	}

	if parsedUser == nil || parsedGroup == nil {
		log.Info("oneagent-nonroot FF present, but failed to parse, ignoring flag", "userRaw", userRaw, "groupRaw", groupRaw)

		return 0, 0, false
	}

	if !*k8ssecuritycontext.IsNonRoot(parsedUser, parsedGroup) {
		log.Info("oneagent-nonroot FF present, but user is root (0), ignoring flag", "parsedUser", *parsedUser, "parsedGroup", *parsedGroup)

		return 0, 0, false
	}

	return *parsedUser, *parsedGroup, true
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
