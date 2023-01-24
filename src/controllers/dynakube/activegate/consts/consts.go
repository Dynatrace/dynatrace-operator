package consts

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	corev1 "k8s.io/api/core/v1"
)

const (
	MultiActiveGateName     = "activegate"
	ActiveGateContainerName = "activegate"
	SyntheticContainerName  = "synthetic"
	ProxySecretSuffix       = "internal-proxy"
	ProxySecretKey          = "proxy"
	HttpsServicePortName    = "https"
	HttpsServicePort        = 443
	HttpServicePortName     = "http"
	HttpServicePort         = 80
	HttpsContainerPort      = 9999
	HttpContainerPort       = 9998

	AuthTokenSecretVolumeName = "ag-authtoken-secret"
	AuthTokenMountPoint       = connectioninfo.TokenBasePath + "/auth-token"

	EnvDtCapabilities    = "DT_CAPABILITIES"
	EnvDtIdSeedNamespace = "DT_ID_SEED_NAMESPACE"
	EnvDtIdSeedClusterId = "DT_ID_SEED_K8S_CLUSTER_ID"
	EnvDtNetworkZone     = "DT_NETWORK_ZONE"
	EnvDtGroup           = "DT_GROUP"
	EnvDtDnsEntryPoint   = "DT_DNS_ENTRY_POINT"

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

var (
	ContainerSecurityContext = corev1.SecurityContext{
		Privileged:               address.Of(false),
		AllowPrivilegeEscalation: address.Of(false),
		RunAsNonRoot:             address.Of(true),
		RunAsUser:                address.Of(DockerImageUser),
		RunAsGroup:               address.Of(DockerImageGroup),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		ReadOnlyRootFilesystem: address.Of(false),
	}

	PodSecurityContext = corev1.PodSecurityContext{
		// Privileged:               address.Of(false),
		// AllowPrivilegeEscalation: address.Of(false),
		RunAsNonRoot: address.Of(true),
		RunAsUser:    address.Of(DockerImageUser),
		RunAsGroup:   address.Of(DockerImageGroup),
		// Capabilities: &corev1.Capabilities{
		//	Drop: []corev1.Capability{
		//		"ALL",
		//	},
		// },
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		//ReadOnlyRootFilesystem: address.Of(false),
	}
)
