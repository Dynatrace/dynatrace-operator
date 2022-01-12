package statsdingest

import "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"

const (
	EecContainerName       = consts.ActiveGateContainerName + "-eec"
	StatsdContainerName    = consts.ActiveGateContainerName + "-statsd"
	StatsdIngestPortName   = "statsd"
	StatsdIngestPort       = 18125
	StatsdIngestTargetPort = "statsd-port"
)
