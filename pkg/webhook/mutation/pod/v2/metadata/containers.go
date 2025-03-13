package metadata

import (
	"strings"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	metacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/metadata"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Mutate(metaClient client.Client, request *dtwebhook.MutationRequest, attributes *podattr.Attributes) bool {
	if !metacommon.IsEnabled(request.BaseRequest) {
		return false
	}

	log.Info("adding metadata-enrichment to pod", "name", request.PodName())

	worloadInfo, err := metacommon.RetrieveWorkload(metaClient, request)
	if err != nil {
		return false // TODO
	}

	attributes.WorkloadInfo = podattr.WorkloadInfo{
		WorkloadKind: worloadInfo.Kind,
		WorkloadName: worloadInfo.Name,
	}

	addMetadataToInitArgs(request, attributes)

	metacommon.SetInjectedAnnotation(request.Pod)
	metacommon.SetWorkloadAnnotations(request.Pod, worloadInfo)

	return true
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

	attributes.UserDefined = metadataAnnotations
}
