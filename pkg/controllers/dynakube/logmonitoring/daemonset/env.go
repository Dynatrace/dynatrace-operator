package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	corev1 "k8s.io/api/core/v1"
)

const (

	// init envs
	nodeNameEnv      = "K8S_NODE_NAME"
	podNameEnv       = "K8S_POD_NAME"
	podUIDEnv        = "K8S_POD_UID"
	namespaceNameEnv = "K8S_NAMESPACE_NAME"
	clusterUIDEnv    = "K8S_CLUSTER_UID"
	clusterNameEnv   = "K8S_CLUSTER_NAME"
	basePodNameEnv   = "K8S_BASEPODNAME"
	entityEnv        = "DT_ENTITY_KUBERNETES_CLUSTER"

	// main container envs
	KubeletNodeNameEnv  = "KUBELET_API_NODENAME"
	KubeletIPAddressEnv = "KUBELET_API_ADDRESS"
	dtStorageEnv        = "DT_STORAGE"
	ruxitConfigEnv      = "APMNG_PA_CONFIG_PATH"

	dtStoragePath   = "/var/lib/dynatrace/oneagent"
	ruxitConfigPath = "/var/lib/dynatrace/oneagent/agent/config/ruxitagentproc.conf"
)

func getInitEnvs(dk dynakube.DynaKube) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: nodeNameEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: podNameEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: podUIDEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
		{
			Name: namespaceNameEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  clusterUIDEnv,
			Value: dk.Status.KubeSystemUUID,
		},
		{
			Name:  clusterNameEnv,
			Value: dk.Status.KubernetesClusterName,
		},
		{
			Name:  entityEnv,
			Value: dk.Status.KubernetesClusterMEID,
		},
		{
			Name:  basePodNameEnv,
			Value: dk.LogMonitoring().GetDaemonSetName(),
		},
	}
}

func GetKubeletEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: KubeletNodeNameEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: KubeletIPAddressEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
	}
}

func getEnvs() []corev1.EnvVar {
	apiEnvs := GetKubeletEnvs()
	standaloneEnvs := []corev1.EnvVar{
		{
			Name:  dtStorageEnv,
			Value: dtStoragePath,
		},
		{
			Name:  ruxitConfigEnv,
			Value: ruxitConfigPath,
		},
	}

	return append(apiEnvs, standaloneEnvs...)
}
