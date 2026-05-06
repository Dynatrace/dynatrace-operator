package attributes

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
)

const (
	K8sNodeNameEnv = "K8S_NODE_NAME"
	K8sPodNameEnv  = "K8S_PODNAME"
	K8sPodUIDEnv   = "K8S_PODUID"

	K8sPodNameAttr       = "k8s.pod.name"
	K8sPodUIDAttr        = "k8s.pod.uid"
	K8sNodeNameAttr      = "k8s.node.name"
	K8sNamespaceNameAttr = "k8s.namespace.name"

	K8sClusterUIDAttr      = "k8s.cluster.uid"
	K8sClusterNameAttr     = "k8s.cluster.name"
	K8sDTClusterEntityAttr = "dt.entity.kubernetes_cluster"
)

func (attrs *PodAttributes) GetPodAttributes(request dtwebhook.BaseRequest) {
	attrs.podEnvVars = append(attrs.podEnvVars, []corev1.EnvVar{
		{Name: K8sPodNameEnv, ValueFrom: k8senv.NewSourceForField("metadata.name")},
		{Name: K8sPodUIDEnv, ValueFrom: k8senv.NewSourceForField("metadata.uid")},
		{Name: K8sNodeNameEnv, ValueFrom: k8senv.NewSourceForField("spec.nodeName")},
	}...)

	attrs.podInfo[K8sPodNameAttr] = createEnvVarRef(K8sPodNameEnv)
	attrs.podInfo[K8sPodUIDAttr] = createEnvVarRef(K8sPodUIDEnv)
	attrs.podInfo[K8sNodeNameAttr] = createEnvVarRef(K8sNodeNameEnv)
	attrs.podInfo[K8sNamespaceNameAttr] = request.Pod.Namespace

	attrs.clusterInfo[K8sClusterUIDAttr] = request.DynaKube.Status.KubeSystemUUID
	attrs.clusterInfo[K8sClusterNameAttr] = request.DynaKube.Status.KubernetesClusterName
	attrs.clusterInfo[K8sDTClusterEntityAttr] = request.DynaKube.Status.KubernetesClusterMEID
}

func createEnvVarRef(envName string) string {
	return fmt.Sprintf("$(%s)", envName)
}
