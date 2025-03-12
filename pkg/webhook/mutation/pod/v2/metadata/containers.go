package metadata

import (
	"context"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	metacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/metadata"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Mutate(context context.Context, metaClient client.Client, request *dtwebhook.MutationRequest) bool {
	if !metacommon.IsEnabled(request.BaseRequest) {
		return false
	}

	log.Info("adding metadata-enrichment to pod", "name", request.PodName())

	// TODO: probably will only modify the iniContainer and pod, not the containers

	return true
}
