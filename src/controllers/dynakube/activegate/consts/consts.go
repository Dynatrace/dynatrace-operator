package consts

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
)

const (
	MultiActiveGateName     = "activegate"
	ActiveGateContainerName = "activegate"
	EecContainerName        = ActiveGateContainerName + "-eec"
	StatsdContainerName     = ActiveGateContainerName + "-statsd"
	SyntheticContainerName  = ActiveGateContainerName + "-synthetic"
	StatsdIngestPort        = 18125
	StatsdIngestTargetPort  = "statsd-port"
	StatsdIngestPortName    = "statsd"
	ProxySecretSuffix       = "internal-proxy"
	ProxySecretKey          = "proxy"
	HttpsServicePortName    = "https"
	HttpsServicePort        = 443
	HttpServicePortName     = "http"
	HttpServicePort         = 80
	HttpsContainerPort      = 9999
	HttpContainerPort       = 9998

	DeploymentTypeActiveGate = "active_gate"

	AuthTokenSecretVolumeName = "ag-authtoken-secret"
	AuthTokenMountPoint       = connectioninfo.TokenBasePath + "/auth-token"

	EnvDtServer             = "DT_SERVER"
	EnvDtTenant             = "DT_TENANT"
	EnvDtCapabilities       = "DT_CAPABILITIES"
	EnvDtIdSeedNamespace    = "DT_ID_SEED_NAMESPACE"
	EnvDtIdSeedClusterId    = "DT_ID_SEED_K8S_CLUSTER_ID"
	EnvDtNetworkZone        = "DT_NETWORK_ZONE"
	EnvDtGroup              = "DT_GROUP"
	EnvDtDeploymentMetadata = "DT_DEPLOYMENT_METADATA"
	EnvDtDnsEntryPoint      = "DT_DNS_ENTRY_POINT"

	AnnotationActiveGateConfigurationHash = dynatracev1beta1.InternalFlagPrefix + "activegate-configuration-hash"
	AnnotationActiveGateContainerAppArmor = "container.apparmor.security.beta.kubernetes.io/" + ActiveGateContainerName

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
