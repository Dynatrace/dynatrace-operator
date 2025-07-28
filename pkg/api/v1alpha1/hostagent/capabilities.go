package hostagent

type CapabilitiesSpec struct {
	KubeMon       KubeMonSpec       `json:"kubernetesMonitoring"`
	MetricsIngest MetricsIngestSpec `json:"metricsIngest"`
	Routing       RoutingSpec       `json:"routing"`
	API           APISpec           `json:"api"`
	Debugging     DebuggingSpec     `json:"debug"`
	EEC           EECSpec           `json:"eec"`
}

type KubeMonSpec struct {
	Enabled                           *bool  `json:"enabled,omitempty"`
	AutomaticApiMonitoring            *bool  `json:"automaticApiMonitoring,omitempty"`
	AutomaticApiMonitoringClusterName string `json:"automaticApiMonitoringClusterName,omitempty"`
	K8sAppEnabled                     *bool  `json:"k8sAppEnabled,omitempty"`
}

type MetricsIngestSpec struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type RoutingSpec struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type APISpec struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type DebuggingSpec struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type EECSpec struct {
	Enabled *bool `json:"enabled,omitempty"`
}
