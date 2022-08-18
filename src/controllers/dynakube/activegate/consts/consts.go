package consts

const (
	MultiActiveGateName     = "activegate"
	ActiveGateContainerName = "activegate"
	EecContainerName        = ActiveGateContainerName + "-eec"
	StatsdContainerName     = ActiveGateContainerName + "-statsd"
	StatsdIngestPort        = 18125
	StatsdIngestTargetPort  = "statsd-port"
	StatsdIngestPortName    = "statsd"
	ProxySecretSuffix       = "internal-proxy"
)
