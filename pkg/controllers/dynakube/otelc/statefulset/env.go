package statefulset

import (
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	otelcConsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	corev1 "k8s.io/api/core/v1"
)

const (
	// default values
	defaultOLTPgrpcPort = "10001"
	defaultOLTPhttpPort = "10002"
	defaultReplicas     = 1

	// env variables
	envShards             = "SHARDS"
	envShardId            = "SHARD_ID"
	envPodNamePrefix      = "POD_NAME_PREFIX"
	envPodName            = "POD_NAME"
	envMyPodIP            = "MY_POD_IP"
	envOTLPgrpcPort       = "OTLP_GRPC_PORT"
	envOTLPhttpPort       = "OTLP_HTTP_PORT"
	envEECDStoken         = "EEC_DS_TOKEN"
	envTrustedCAs         = "TRUSTED_CAS"
	envK8sClusterName     = "K8S_CLUSTER_NAME"
	envK8sClusterUid      = "K8S_CLUSTER_UID"
	envDTentityK8sCluster = "DT_ENTITY_KUBERNETES_CLUSTER"
	envDTendpoint         = "DT_ENDPOINT"
	// certDirEnv is the environment variable that identifies which directory
	// to check for SSL certificate files. If set, this overrides the system default.
	// It is a colon separated list of directories.
	// See https://www.openssl.org/docs/man1.0.2/man1/c_rehash.html.
	envCertDir          = "SSL_CERT_DIR"
	envEECcontrollerTLS = "EXTENSIONS_CONTROLLER_TLS"
	envHttpProxy        = "HTTP_PROXY"
	envHttpsProxy       = "HTTPS_PROXY"
	envNoProxy          = "NO_PROXY"

	// Volume names and paths
	customEecTLSCertificatePath     = "/tls/custom/eec"
	customEecTLSCertificateFullPath = customEecTLSCertificatePath + "/" + consts.TLSCrtDataName
)

func getEnvs(dk *dynakube.DynaKube) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{Name: envShards, Value: strconv.Itoa(int(getReplicas(dk)))},
		{Name: envPodNamePrefix, Value: dk.OtelCollectorStatefulsetName()},
		{Name: envPodName, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['statefulset.kubernetes.io/pod-name']",
			},
		},
		},
		{Name: envShardId, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['apps.kubernetes.io/pod-index']",
			},
		},
		},
		{Name: envOTLPgrpcPort, Value: defaultOLTPgrpcPort},
		{Name: envOTLPhttpPort, Value: defaultOLTPhttpPort},
		{Name: envK8sClusterName, Value: dk.Name},
		{Name: envK8sClusterUid, Value: dk.Status.KubeSystemUUID},
		{Name: envDTentityK8sCluster, Value: dk.Status.KubernetesClusterMEID},
	}

	if dk.HasProxy() {
		envs = append(envs, getDynakubeProxyEnvValue(envHttpsProxy, dk.Spec.Proxy))
		envs = append(envs, getDynakubeProxyEnvValue(envHttpProxy, dk.Spec.Proxy))
		envs = append(envs, corev1.EnvVar{Name: envNoProxy, Value: getDynakubeNoProxyEnvValue(dk)})
	}

	if dk.IsExtensionsEnabled() {
		envs = append(
			envs,
			corev1.EnvVar{Name: envEECDStoken, ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: dk.ExtensionsTokenSecretName()},
					Key:                  consts.OtelcTokenSecretKey,
				}},
			},
			corev1.EnvVar{Name: envCertDir, Value: customEecTLSCertificatePath},
			corev1.EnvVar{Name: envEECcontrollerTLS, Value: customEecTLSCertificateFullPath},
		)
	}

	if dk.TelemetryIngest().IsEnabled() {
		envs = append(envs,
			corev1.EnvVar{Name: envDTendpoint, ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: otelcConsts.OtlpApiEndpointConfigMapName},
					Key:                  envDTendpoint,
				},
			}},
			corev1.EnvVar{Name: envMyPodIP, ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			}},
			corev1.EnvVar{Name: otelcConsts.EnvDataIngestToken, ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: dk.Tokens()},
					Key:                  dynatrace.DataIngestToken,
				},
			}},
		)
	}

	if dk.IsExtensionsEnabled() && dk.Spec.TrustedCAs != "" {
		envs = append(envs, corev1.EnvVar{Name: envTrustedCAs, Value: otelcConsts.TrustedCAVolumePath})
	}

	return envs
}

func getDynakubeProxyEnvValue(envVar string, src *value.Source) corev1.EnvVar {
	if src.ValueFrom != "" {
		return corev1.EnvVar{
			Name: envVar,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: src.ValueFrom},
					Key:                  dynakube.ProxyKey,
				},
			},
		}
	}

	return corev1.EnvVar{Name: envVar, Value: src.Value}
}

func getDynakubeNoProxyEnvValue(dk *dynakube.DynaKube) string {
	noProxyValues := []string{}

	if dk.IsExtensionsEnabled() {
		noProxyValues = append(noProxyValues, dk.ExtensionsServiceNameFQDN())
	}

	if dk.ActiveGate().IsEnabled() {
		noProxyValues = append(noProxyValues, capability.BuildServiceName(dk.Name)+"."+dk.Namespace)
	}

	return strings.Join(noProxyValues, ",")
}
