// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package consts

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	corev1 "k8s.io/api/core/v1"
)

const (
	MultiActiveGateName     = "activegate"
	ActiveGateContainerName = "activegate"

	HTTPSServicePortName = "https"
	HTTPSServicePort     = 443
	HTTPSContainerPort   = 9999
	HTTPServicePortName  = "http"
	HTTPServicePort      = 80
	HTTPContainerPort    = 9998

	AnnotationActiveGateConfigurationHash = api.InternalFlagPrefix + "activegate-configuration-hash"
	AnnotationActiveGateTenantTokenHash   = api.InternalFlagPrefix + "activegate-tenant-token-hash"
	AnnotationActiveGateContainerAppArmor = corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + ActiveGateContainerName

	EnvDTCapabilities    = "DT_CAPABILITIES"
	EnvDTIDSeedNamespace = "DT_ID_SEED_NAMESPACE"
	EnvDTIDSeedClusterID = "DT_ID_SEED_K8S_CLUSTER_ID"
	EnvDTNetworkZone     = "DT_NETWORK_ZONE"
	EnvDTGroup           = "DT_GROUP"
	EnvDTDNSEntryPoint   = "DT_DNS_ENTRY_POINT"
	EnvDTHTTPPort        = "DT_HTTP_PORT"

	// Volumes: gateway filesystem (readonly modifier)
	GatewayConfigVolumeName  = "ag-lib-gateway-config"
	GatewayConfigMountPath   = "/var/lib/dynatrace/gateway/config"
	GatewayLibTempVolumeName = "ag-lib-gateway-temp"
	GatewayLibTempMountPath  = "/var/lib/dynatrace/gateway/temp"
	GatewayDataVolumeName    = "ag-lib-gateway-data"
	GatewayDataMountPath     = "/var/lib/dynatrace/gateway/data"
	GatewaySslVolumeName     = "ag-lib-gateway-ssl"
	GatewaySslMountPath      = "/var/lib/dynatrace/gateway/ssl"
	GatewayLogVolumeName     = "ag-log-gateway"
	GatewayLogMountPath      = "/var/log/dynatrace/gateway"
	GatewayTmpVolumeName     = "ag-tmp-gateway"
	GatewayTmpMountPath      = "/var/tmp/dynatrace/gateway"

	// Volumes: auth token
	AuthTokenSecretVolumeName = "ag-authtoken-secret"
	AuthTokenMountPoint       = consts.DTComponentsSecretsRootDir + "/tokens/auth-token"

	// Volumes: tenant token (connection info)
	TenantSecretVolumeName = "connection-info-secret"
	TenantTokenMountPath   = consts.DTComponentsSecretsRootDir + "/tokens/tenant-token"

	// Volumes: TLS certificates
	CertsVolumeName = "server-certs"
	CertsMountPath  = consts.DTComponentsSecretsRootDir + "/tls"

	// Volumes: trusted CAs
	TrustedCAsVolumeName = "trustedcas"
	TrustedCAsMountPath  = consts.DTComponentsSecretsRootDir + "/rootca"

	// Volumes: custom properties
	CustomPropertiesVolumeName = "custom-properties"
	CustomPropertiesMountPath  = "/var/lib/dynatrace/gateway/config_template/custom.properties"

	// Volumes: deployment properties
	DeploymentPropertiesVolumeName = "deployment-properties"
	DeploymentPropertiesFileName   = "deployment.properties"
	DeploymentPropertiesBasePath   = consts.DTComponentsSecretsRootDir + "/config"
	DeploymentPropertiesMountPath  = DeploymentPropertiesBasePath + "/" + DeploymentPropertiesFileName

	// Volumes: internal proxy
	ProxySecretVolumeName = proxy.SecretVolumeName
	ProxySecretMountPath  = proxy.SecretMountPath

	// Volumes: EEC (Extensions Execution Controller)
	EECVolumeName = "eec-token"
	EECMountPath  = consts.DTComponentsSecretsRootDir + "/eec/token"

	// Volumes: KSPM token
	KSPMTokenVolumeName = "kspm-token"
	KSPMTokenMountPath  = consts.DTComponentsSecretsRootDir + "/tokens/kspm/node-configuration-collector"

	// Volumes: Kubernetes monitoring truststore
	TrustStoreVolumeName       = "truststore-volume"
	TrustStoreCacertsMountPath = "/opt/dynatrace/gateway/jre/lib/security/cacerts"

	// Volumes: certificate-loader init container working directory
	InitCertLoaderWorkDirVolumeName = "cert-tmp"
	InitCertLoaderWorkDirMountPath  = "/var/lib/dynatrace/gateway"

	DockerImageUser  int64 = 1001
	DockerImageGroup int64 = 1001
)

var (
	VolumeNames = []string{
		AuthTokenSecretVolumeName,
		TenantSecretVolumeName,
		CertsVolumeName,
		TrustedCAsVolumeName,
		CustomPropertiesVolumeName,
		DeploymentPropertiesVolumeName,
		ProxySecretVolumeName,
		EECVolumeName,
		KSPMTokenVolumeName,
		TrustStoreVolumeName,
		InitCertLoaderWorkDirVolumeName,
		GatewayConfigVolumeName,
		GatewayLibTempVolumeName,
		GatewayDataVolumeName,
		GatewaySslVolumeName,
		GatewayLogVolumeName,
		GatewayTmpVolumeName,
	}

	VolumeMountPaths = []string{
		AuthTokenMountPoint,
		TenantTokenMountPath,
		CertsMountPath,
		TrustedCAsMountPath,
		CustomPropertiesMountPath,
		DeploymentPropertiesMountPath,
		ProxySecretMountPath,
		EECMountPath,
		KSPMTokenMountPath,
		TrustStoreCacertsMountPath,
		GatewayConfigMountPath,
		GatewayLibTempMountPath,
		GatewayDataMountPath,
		GatewaySslMountPath,
		GatewayLogMountPath,
		GatewayTmpMountPath,
		// Only part of the init-container and is rather top level directory
		// InitCertLoaderWorkDirMountPath,
	}
)
