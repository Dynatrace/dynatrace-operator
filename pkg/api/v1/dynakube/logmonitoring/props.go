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

func (lm *LogMonitoring) GetNodeSelector() map[string]string {
	if lm.IsStandalone() && lm.TemplateSpec != nil {
		return lm.NodeSelector
	}

	return nil
}

// Template is a nil-safe way to access the underlying TemplateSpec.
func (lm *LogMonitoring) Template() TemplateSpec {
	if lm.TemplateSpec == nil {
		return TemplateSpec{}
	}

	return *lm.TemplateSpec
}
