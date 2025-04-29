package metadata

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	metacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/metadata"
	corev1 "k8s.io/api/core/v1"
)

func propagateMetadataAnnotations(request *dtwebhook.MutationRequest) {
	metacommon.CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
	addMetadataToInitEnv(request.Pod, request.InstallContainer)
}

func addMetadataToInitEnv(pod *corev1.Pod, installContainer *corev1.Container) {
	if value, ok := pod.Annotations[dynakube.MetadataAnnotation]; ok {
		installContainer.Env = append(installContainer.Env,
			corev1.EnvVar{
				Name: consts.EnrichmentWorkloadAnnotationsEnv, Value: value},
		)
	}
}
