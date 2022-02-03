package consts

const (
	ActiveGateContainerName = "activegate"

	HttpsServicePortName = "https"
	HttpsServicePort     = 443
	HttpServicePortName  = "http"
	HttpServicePort      = 80

	EecContainerName = ActiveGateContainerName + "-eec"

	StatsDContainerName    = ActiveGateContainerName + "-statsd"
	StatsDIngestPortName   = "statsd"
	StatsDIngestPort       = 18125
	StatsDIngestTargetPort = "statsd-port"
)
