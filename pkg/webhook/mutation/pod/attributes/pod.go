package attributes

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"

	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	"k8s.io/api/core/v1"
)

func GetPodAttributes(attrs pod.Attributes, request *mutator.BaseRequest) (pod.Attributes, []v1.EnvVar) {

	attrs.PodInfo = pod.PodInfo{
		PodName:       createEnvVarRef(K8sPodNameEnv),
		PodUID:        createEnvVarRef(K8sPodUIDEnv),
		NodeName:      createEnvVarRef(K8sNodeNameEnv),
		NamespaceName: request.Pod.Namespace,
	}

	attrs.ClusterInfo = pod.ClusterInfo{
		ClusterUID:      request.DynaKube.Status.KubeSystemUUID,
		DTClusterEntity: request.DynaKube.Status.KubernetesClusterMEID,
		ClusterName:     request.DynaKube.Status.KubernetesClusterName,
	}

	setDeprecatedAttributes(attrs)

	envs := []v1.EnvVar{
		{Name: K8sPodNameEnv, ValueFrom: k8senv.NewSourceForField("metadata.name")},
		{Name: K8sPodUIDEnv, ValueFrom: k8senv.NewSourceForField("metadata.uid")},
		{Name: K8sNodeNameEnv, ValueFrom: k8senv.NewSourceForField("spec.nodeName")},
	}

	return attrs, envs
}

func GetContainerAttributes(request *mutator.MutationRequest, containers []*v1.Container) []container.Attributes {
	attributes := []container.Attributes{}
	for _, c := range containers {
		attributes = append(attributes, container.Attributes{
			ImageInfo:     createImageInfo(c.Image),
			ContainerName: c.Name,
		})

		volumes.AddConfigVolumeMount(c, request.BaseRequest)
	}

	return attributes
}

func createEnvVarRef(envName string) string {
	return fmt.Sprintf("$(%s)", envName)
}
