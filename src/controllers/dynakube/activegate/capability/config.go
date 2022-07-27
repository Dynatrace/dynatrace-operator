package capability

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("activegate-capability")
)

const (
	ActiveGateContainerName = "activegate"

	ActiveGateGatewayConfigVolumeName = "ag-lib-gateway-config"
	ActiveGateGatewayTempVolumeName   = "ag-lib-gateway-temp"
	ActiveGateGatewayDataVolumeName   = "ag-lib-gateway-data"
	ActiveGateGatewaySslVolumeName    = "ag-lib-gateway-ssl"
	ActiveGateLogVolumeName           = "ag-log-gateway"
	ActiveGateTmpVolumeName           = "ag-tmp-gateway"

	ActiveGateGatewayConfigMountPoint = "/var/lib/dynatrace/gateway/config"
	ActiveGateGatewayTempMountPoint   = "/var/lib/dynatrace/gateway/temp"
	ActiveGateGatewayDataMountPoint   = "/var/lib/dynatrace/gateway/data"
	ActiveGateGatewaySslMountPoint    = "/var/lib/dynatrace/gateway/ssl"
	ActiveGateLogMountPoint           = "/var/log/dynatrace/gateway"
	ActiveGateTmpMountPoint           = "/var/tmp/dynatrace/gateway"

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
