package logmonitoring

const (
	logModuleDaemonSetSuffix = "-logmodule"
)

func (logMonitoring *Spec) SetName(name string) {
	logMonitoring.name = name
}

func (logMonitoring *Spec) Needed() bool {
	return logMonitoring.Enabled
}

func (logMonitoring *Spec) GetDaemonSetName() string {
	return logMonitoring.name + logModuleDaemonSetSuffix
}

func (template *TemplateSpec) GetLogModuleNodeSelector() map[string]string {
	return template.NodeSelector
}
