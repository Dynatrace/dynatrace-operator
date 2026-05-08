package attributes

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
)

func (attrs *PodAttributes) readPodAttributes(request dtwebhook.BaseRequest) {
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
