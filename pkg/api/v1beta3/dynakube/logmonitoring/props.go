package logmonitoring

const (
	logMonitoringDaemonSetSuffix = "-logmonitoring"
)

func (logMonitoring *Spec) SetName(name string) {
	logMonitoring.name = name
}

func (logMonitoring *Spec) Needed() bool {
	return logMonitoring.Enabled
}

func (logMonitoring *Spec) GetDaemonSetName() string {
	return logMonitoring.name + logMonitoringDaemonSetSuffix
}
