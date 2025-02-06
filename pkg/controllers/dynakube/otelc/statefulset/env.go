package statefulset

import (
	"fmt"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
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
	envOTLPgrpcPort       = "OTLP_GRPC_PORT"
	envOTLPhttpPort       = "OTLP_HTTP_PORT"
	envEECDStoken         = "EEC_DS_TOKEN"
	envTrustedCAs         = "TRUSTED_CAS"
	envK8sClusterName     = "K8S_CLUSTER_NAME"
	envK8sClusterUid      = "K8S_CLUSTER_UID"
	envDTentityK8sCluster = "DT_ENTITY_KUBERNETES_CLUSTER"
	// certDirEnv is the environment variable that identifies which directory
	// to check for SSL certificate files. If set, this overrides the system default.
	// It is a colon separated list of directories.
	// See https://www.openssl.org/docs/man1.0.2/man1/c_rehash.html.
	envCertDir          = "SSL_CERT_DIR"
	envEECcontrollerTLS = "EXTENSIONS_CONTROLLER_TLS"
	envDTEndpoint       = "DT_ENDPOINT"
	envDTApiToken       = "DT_API_TOKEN"
	envMyPodIP          = "MY_POD_IP"

	// Volume names and paths
	trustedCAVolumeMountPath        = "/tls/custom/cacerts"
	trustedCAVolumePath             = trustedCAVolumeMountPath + "/certs"
	customEecTLSCertificatePath     = "/tls/custom/eec"
	customEecTLSCertificateFullPath = customEecTLSCertificatePath + "/" + consts.TLSCrtDataName
)

func getEnvs(dk *dynakube.DynaKube) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{Name: envShards, Value: strconv.Itoa(int(getReplicas(dk)))},
		{Name: envPodNamePrefix, Value: dk.ExtensionsCollectorStatefulsetName()},
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
	if dk.Spec.TrustedCAs != "" {
		envs = append(envs, corev1.EnvVar{Name: envTrustedCAs, Value: trustedCAVolumePath})
	}

	if dk.IsExtensionsEnabled() {
		envs = append(envs,
			corev1.EnvVar{
				Name:  envCertDir,
				Value: customEecTLSCertificatePath,
			},
			corev1.EnvVar{
				Name: envEECDStoken,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: dk.ExtensionsTokenSecretName()},
						Key:                  consts.OtelcTokenSecretKey,
					},
				},
			},
			corev1.EnvVar{
				Name:  envEECcontrollerTLS,
				Value: customEecTLSCertificateFullPath,
			})
	}

	if !dk.IsExtensionsEnabled() {
		envs = append(envs,
			corev1.EnvVar{
				Name:  envDTEndpoint,
				Value: buildDTEndpoint(dk),
			})

		envs = append(envs,
			corev1.EnvVar{
				Name: envDTApiToken,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: dk.Tokens()},
						Key:                  dynatrace.ApiToken,
					},
				},
			})

		envs = append(envs,
			corev1.EnvVar{
				Name: envMyPodIP,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "",
						FieldPath:  "status.podIP",
					},
				},
			})
	}

	return envs
}

func buildDTEndpoint(dk *dynakube.DynaKube) string {
	if dk.ActiveGate().IsEnabled() {
		serviceName := capability.BuildServiceName(dk.Name, agconsts.MultiActiveGateName)

		return fmt.Sprintf("https://%s.%s/api", serviceName, dk.Namespace)
	}

	return dk.Spec.APIURL
}
