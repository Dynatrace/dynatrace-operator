package consts

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
)

const (
	MultiActiveGateName     = "activegate"
	ActiveGateContainerName = "activegate"
	ProxySecretSuffix       = "internal-proxy"
	HTTPSServicePortName    = "https"
	HTTPSServicePort        = 443
	HTTPSContainerPort      = 9999
	HTTPServicePortName     = "http"
	HTTPServicePort         = 80
	HTTPContainerPort       = 9998

	AuthTokenSecretVolumeName = "ag-authtoken-secret"
	AuthTokenMountPoint       = connectioninfo.TokenBasePath + "/auth-token"

	EnvDtCapabilities    = "DT_CAPABILITIES"
	EnvDtIDSeedNamespace = "DT_ID_SEED_NAMESPACE"
	EnvDtIDSeedClusterID = "DT_ID_SEED_K8S_CLUSTER_ID"
	EnvDtNetworkZone     = "DT_NETWORK_ZONE"
	EnvDtGroup           = "DT_GROUP"
	EnvDtDNSEntryPoint   = "DT_DNS_ENTRY_POINT"
	EnvDtHTTPPort        = "DT_HTTP_PORT"

	AnnotationActiveGateConfigurationHash = api.InternalFlagPrefix + "activegate-configuration-hash"
	AnnotationActiveGateTenantTokenHash   = api.InternalFlagPrefix + "activegate-tenant-token-hash"
	AnnotationActiveGateContainerAppArmor = "container.apparmor.security.beta.kubernetes.io/" + ActiveGateContainerName

	GatewayConfigVolumeName  = "ag-lib-gateway-config"
	GatewayLibTempVolumeName = "ag-lib-gateway-temp"
	GatewayDataVolumeName    = "ag-lib-gateway-data"
	GatewaySslVolumeName     = "ag-lib-gateway-ssl"
	GatewayLogVolumeName     = "ag-log-gateway"
	GatewayTmpVolumeName     = "ag-tmp-gateway"
	GatewayConfigMountPoint  = "/var/lib/dynatrace/gateway/config"
	GatewayLibTempMountPoint = "/var/lib/dynatrace/gateway/temp"
	GatewayDataMountPoint    = "/var/lib/dynatrace/gateway/data"
	GatewaySslMountPoint     = "/var/lib/dynatrace/gateway/ssl"
	GatewayLogMountPoint     = "/var/log/dynatrace/gateway"
	GatewayTmpMountPoint     = "/var/tmp/dynatrace/gateway"

	DockerImageUser  int64 = 1001
	DockerImageGroup int64 = 1001
)
