package activegate

type CapabilityDisplayName string

type Capability struct {
	// The name of the capability known by the user, mainly used in the CR
	DisplayName CapabilityDisplayName

	// The name used for marking the pod for given capability
	ShortName string

	// The string passed to the active gate image to enable a given capability
	ArgumentName string
}

var (
	RoutingCapability = Capability{
		DisplayName:  "routing",
		ShortName:    "routing",
		ArgumentName: "MSGrouter",
	}

	KubeMonCapability = Capability{
		DisplayName:  "kubernetes-monitoring",
		ShortName:    "kubemon",
		ArgumentName: "kubernetes_monitoring",
	}

	MetricsIngestCapability = Capability{
		DisplayName:  "metrics-ingest",
		ShortName:    "metrics-ingest",
		ArgumentName: "metrics_ingest",
	}

	DynatraceApiCapability = Capability{
		DisplayName:  "dynatrace-api",
		ShortName:    "dynatrace-api",
		ArgumentName: "restInterface",
	}
)

var ActiveGateDisplayNames = map[CapabilityDisplayName]struct{}{
	RoutingCapability.DisplayName:       {},
	KubeMonCapability.DisplayName:       {},
	MetricsIngestCapability.DisplayName: {},
	DynatraceApiCapability.DisplayName:  {},
}
