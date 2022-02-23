package capability

const (
	ActiveGateContainerName = "activegate"

	HttpsServicePortName = "https"
	HttpsServicePort     = 443
	HttpServicePortName  = "http"
	HttpServicePort      = 80

	EecContainerName       = ActiveGateContainerName + "-eec"
	StatsdContainerName    = ActiveGateContainerName + "-statsd"
	StatsdIngestPortName   = "statsd"
	StatsdIngestPort       = 18125
	StatsdIngestTargetPort = "statsd-port"
)
