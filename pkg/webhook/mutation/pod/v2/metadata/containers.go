package metadata

import (
	"maps"
	"strings"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	metacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/metadata"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Mutate(metaClient client.Client, request *dtwebhook.MutationRequest, attributes *podattr.Attributes) error {
	log.Info("adding metadata-enrichment to pod", "name", request.PodName())

	workloadInfo, err := metacommon.RetrieveWorkload(metaClient, request)
	if err != nil {
		return err
	}

	attributes.WorkloadInfo = podattr.WorkloadInfo{
		WorkloadKind: workloadInfo.Kind,
		WorkloadName: workloadInfo.Name,
	}

	addMetadataToInitArgs(request, attributes)

	metacommon.SetInjectedAnnotation(request.Pod)
	metacommon.SetWorkloadAnnotations(request.Pod, workloadInfo)

	return nil
}

func addMetadataToInitArgs(request *dtwebhook.MutationRequest, attributes *podattr.Attributes) {
	metacommon.CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)

	metadataAnnotations := map[string]string{}

	for key, value := range request.Pod.Annotations {
		if !strings.HasPrefix(key, dynakube.MetadataPrefix) {
			continue
		}

		split := strings.Split(key, dynakube.MetadataPrefix)
		metadataAnnotations[split[1]] = value
	}

	if attributes.UserDefined == nil {
		attributes.UserDefined = map[string]string{}
	}

	maps.Copy(attributes.UserDefined, metadataAnnotations)
}
