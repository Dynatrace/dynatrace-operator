package exp

const (
	// Deprecated: AGDisableUpdatesKey use AnnotationFeatureActiveGateUpdates instead.
	AGDisableUpdatesKey = FFPrefix + "disable-activegate-updates"
	// Deprecated: Use NoProxy instead.
	AGIgnoreProxyKey = FFPrefix + "activegate-ignore-proxy"

	AGUpdatesKey                              = FFPrefix + "activegate-updates"
	AGAppArmorKey                             = FFPrefix + "activegate-apparmor"
	AGAutomaticK8sApiMonitoringKey            = FFPrefix + "automatic-kubernetes-api-monitoring"
	AGAutomaticK8sApiMonitoringClusterNameKey = FFPrefix + "automatic-kubernetes-api-monitoring-cluster-name"
	AGK8sAppEnabledKey                        = FFPrefix + "k8s-app-enabled"
	AGAutomaticTLSCertificateKey              = FFPrefix + "automatic-tls-certificate"
)

// IsActiveGateUpdatesDisabled is a feature flag to disable ActiveGate updates.
func (ff *FeatureFlags) IsActiveGateUpdatesDisabled() bool {
	return ff.getDisableFlagWithDeprecatedAnnotation(AGUpdatesKey, AGDisableUpdatesKey)
}

// IsActiveGateAutomaticTLSCertificate is a feature flag to disable automatic creation of ActiveGate TLS certificate.
func (ff *FeatureFlags) IsActiveGateAutomaticTLSCertificate() bool {
	return ff.getFeatureFlagBool(AGAutomaticTLSCertificateKey, true)
}

// IsAutomaticK8sApiMonitoring is a feature flag to enable automatic kubernetes api monitoring,
// which ensures that settings for this kubernetes cluster exist in Dynatrace.
func (ff *FeatureFlags) IsAutomaticK8sApiMonitoring() bool {
	return ff.getFeatureFlagBool(AGAutomaticK8sApiMonitoringKey, true)
}

// GetAutomaticK8sApiMonitoringClusterName is a feature flag to set custom cluster name for automatic-kubernetes-api-monitoring.
func (ff *FeatureFlags) GetAutomaticK8sApiMonitoringClusterName() string {
	return ff.getFeatureFlagRaw(AGAutomaticK8sApiMonitoringClusterNameKey)
}

// IsK8sAppEnabled is a feature flag to enable automatically enable current Kubernetes cluster for the Kubernetes app.
func (ff *FeatureFlags) IsK8sAppEnabled() bool {
	return ff.getFeatureFlagBool(AGK8sAppEnabledKey, false)
}

// IsActiveGateAppArmor is a feature flag to enable AppArmor in ActiveGate container.
func (ff *FeatureFlags) IsActiveGateAppArmor() bool {
	return ff.getFeatureFlagBool(AGAppArmorKey, false)
}

// Deprecated: Use NoProxy annotation instead.
// AGIgnoresProxy is a feature flag to ignore the proxy for ActiveGate when set in CR.
func (ff *FeatureFlags) AGIgnoresProxy() bool {
	return ff.getFeatureFlagBool(AGIgnoreProxyKey, false)
}
