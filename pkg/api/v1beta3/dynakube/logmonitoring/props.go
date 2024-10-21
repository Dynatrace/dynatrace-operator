package logmonitoring

const (
	logMonitoringDaemonSetSuffix = "-logmonitoring"
)

func (logMonitoring *LogMonitoring) SetName(name string) {
	logMonitoring.name = name
}

func (logMonitoring *LogMonitoring) IsEnabled() bool {
	return logMonitoring.Spec != nil
}

func (logMonitoring *LogMonitoring) GetDaemonSetName() string {
	return logMonitoring.name + logMonitoringDaemonSetSuffix
}
