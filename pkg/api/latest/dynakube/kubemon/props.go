package kubemon

const KubeMonAvailableConditionType = "KubernetesMonitoringAvailable"

func (km *Spec) IsEnabled() bool {
	return km != nil
}
