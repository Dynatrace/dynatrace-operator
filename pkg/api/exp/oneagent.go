package exp

const (
	// Deprecated: OAProxyIgnoredKey use NoProxy annotation instead.
	OAProxyIgnoredKey = FFPrefix + "oneagent-ignore-proxy"

	OAMaxUnavailableKey      = FFPrefix + "oneagent-max-unavailable"
	OAInitialConnectRetryKey = FFPrefix + "oneagent-initial-connect-retry-ms"
	OAPrivilegedKey          = FFPrefix + "oneagent-privileged"
	OASkipLivenessProbeKey   = FFPrefix + "oneagent-skip-liveness-probe"

	OANodeImagePullKey = FFPrefix + "node-image-pull"
	// OANodeImagePullTechnologiesKey can be set on a Pod or DynaKube to configure which code module technologies to download. It's set to
	// "all" if not set.
	OANodeImagePullTechnologiesKey = "oneagent.dynatrace.com/technologies"
)

const (
	DefaultOAIstioInitialConnectRetry = 6000
)

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

func (ff *FeatureFlags) SkipOneAgentLivenessProbe() bool {
	return ff.getBoolWithDefault(OASkipLivenessProbeKey, false)
}

func (ff *FeatureFlags) IsNodeImagePull() bool {
	return ff.getBoolWithDefault(OANodeImagePullKey, false)
}

func (ff *FeatureFlags) GetNodeImagePullTechnology() string {
	return ff.getRaw(OANodeImagePullTechnologiesKey)
}
