package metadata

import (
	"encoding/json"
	"strings"

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
	metadataAnnotations := map[string]string{}

	for key, value := range pod.Annotations {
		if strings.EqualFold(key, dynakube.MetadataAnnotation) {
			var unmarshaledMetadata map[string]string
			err := json.Unmarshal([]byte(value), &unmarshaledMetadata)
			if err != nil {
				log.Error(err, "failed to unmarshal metadata annotation", "key", key, "value", value)
			}

			for key, value := range unmarshaledMetadata {
				metadataAnnotations[key] = value
			}
		} else if strings.HasPrefix(key, dynakube.MetadataPrefix) {
			split := strings.Split(key, dynakube.MetadataPrefix)
			metadataAnnotations[split[1]] = value
		}
	}

	workloadAnnotationsJson, _ := json.Marshal(metadataAnnotations)
	installContainer.Env = append(installContainer.Env,
		corev1.EnvVar{
			Name: consts.EnrichmentWorkloadAnnotationsEnv, Value: string(workloadAnnotationsJson)},
	)
}
