package exp

const (
	ActiveGateUpdatesAnnotation = AnnotationPrefix + "activegate-updates"

	ActiveGateAppArmorAnnotation                = AnnotationPrefix + "activegate-apparmor"
	AutomaticK8sApiMonitoringAnnotation         = AnnotationPrefix + "automatic-kubernetes-api-monitoring"
	CustomK8sApiMonitoringClusterNameAnnotation = AnnotationPrefix + "automatic-kubernetes-api-monitoring-cluster-name"
	K8sAppEnabledAnnotation                     = AnnotationPrefix + "k8s-app-enabled"
)

// FeatureDisableActiveGateUpdates is a feature flag to disable ActiveGate updates.
func (f *FeatureFlags) IsActiveGateUpdatesDisabled() bool {
	return f.getDisableFlagWithDeprecatedAnnotation(ActiveGateUpdatesAnnotation, DisableActiveGateUpdatesAnnotation)
}

// FeatureAutomaticKubernetesApiMonitoring is a feature flag to enable automatic kubernetes api monitoring,
// which ensures that settings for this kubernetes cluster exist in Dynatrace.
func (f *FeatureFlags) IsAutomaticKubernetesApiMonitoringEnabled() bool {
	return f.getRawValue(AutomaticK8sApiMonitoringAnnotation) != falsePhrase
}

// FeatureAutomaticKubernetesApiMonitoringClusterName is a feature flag to set custom cluster name for automatic-kubernetes-api-monitoring.
func (f *FeatureFlags) GetCustomK8sApiMonitoringClusterName() string {
	return f.getRawValue(CustomK8sApiMonitoringClusterNameAnnotation)
}

// FeatureEnableK8sAppEnabled is a feature flag to enable automatically enable current Kubernetes cluster for the Kubernetes app.
func (f *FeatureFlags) ShouldEnableK8sApp() bool {
	return f.getRawValue(K8sAppEnabledAnnotation) == truePhrase
}

// FeatureActiveGateAppArmor is a feature flag to enable AppArmor in ActiveGate container.
func (f *FeatureFlags) IsActiveGateAppArmorEnabled() bool {
	return f.getRawValue(ActiveGateAppArmorAnnotation) == truePhrase
}
