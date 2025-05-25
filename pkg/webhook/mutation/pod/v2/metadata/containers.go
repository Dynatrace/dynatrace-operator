package metadata

import (
	"maps"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
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
	copiedMetadataAnnotations := metacommon.CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
	if copiedMetadataAnnotations == nil {
		log.Info("copied metadata annotations from namespace is nil, not copying map to attributes UserDefined")

		return
	}

	if attributes.UserDefined == nil {
		attributes.UserDefined = make(map[string]string)
	}

	maps.Copy(attributes.UserDefined, copiedMetadataAnnotations)
}
