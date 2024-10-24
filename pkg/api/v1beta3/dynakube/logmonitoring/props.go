package logmonitoring

const (
	daemonSetSuffix = "-logmonitoring"
)

func (lm *LogMonitoring) SetName(name string) {
	lm.name = name
}

func (lm *LogMonitoring) SetHostAgentDependency(isEnabled bool) {
	lm.enabledDependencies.hostAgents = isEnabled
}

func (lm *LogMonitoring) IsEnabled() bool {
	return lm.Spec != nil
}

func (lm *LogMonitoring) GetDaemonSetName() string {
	return lm.name + daemonSetSuffix
}

func (lm *LogMonitoring) IsStandalone() bool {
	return lm.IsEnabled() && !lm.enabledDependencies.hostAgents
}
