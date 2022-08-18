package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	InternalProxySecretMountPath = "/var/lib/dynatrace/secrets/internal-proxy"

	InternalProxySecretVolumeName = "internal-proxy-secret-volume"

	InternalProxySecretHost          = "host"
	InternalProxySecretHostMountPath = InternalProxySecretMountPath + "/" + InternalProxySecretHost

	InternalProxySecretPort          = "port"
	InternalProxySecretPortMountPath = InternalProxySecretMountPath + "/" + InternalProxySecretPort

	InternalProxySecretUsername          = "username"
	InternalProxySecretUsernameMountPath = InternalProxySecretMountPath + "/" + InternalProxySecretUsername

	InternalProxySecretPassword          = "password"
	InternalProxySecretPasswordMountPath = InternalProxySecretMountPath + "/" + InternalProxySecretPassword

	GatewayConfigVolumeName = "ag-lib-gateway-config"
	GatewayTempVolumeName   = "ag-lib-gateway-temp"
	GatewayDataVolumeName   = "ag-lib-gateway-data"
	GatewaySslVolumeName    = "ag-lib-gateway-ssl"
	LogVolumeName           = "ag-log-gateway"
	TmpVolumeName           = "ag-tmp-gateway"
	GatewayConfigMountPoint = "/var/lib/dynatrace/gateway/config"
	GatewayTempMountPoint   = "/var/lib/dynatrace/gateway/temp"
	GatewayDataMountPoint   = "/var/lib/dynatrace/gateway/data"
	GatewaySslMountPoint    = "/var/lib/dynatrace/gateway/ssl"
	LogMountPoint           = "/var/log/dynatrace/gateway"
	TmpMountPoint           = "/var/tmp/dynatrace/gateway"
)

var (
	log = logger.NewDTLogger().WithName("dynakube-activegate-statefulsetreconciler")
)
