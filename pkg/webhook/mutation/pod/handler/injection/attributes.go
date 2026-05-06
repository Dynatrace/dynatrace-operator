package injection

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
)

func addContainerAttributes(request *dtwebhook.MutationRequest) (bool, error) {
	containers := request.NewContainers(isInjected)
	if len(containers) > 0 {
		args := make([]string, 0)

		for _, c := range containers {

			contInfos := *attributes.NewContainerInfos(*c)

			json, err := contInfos.ToJson()
			if err != nil {
				return false, err
			}

			args = append(args, fmt.Sprintf("--%s=%s", container.Flag, json))

			volumes.AddConfigVolumeMount(c, request.BaseRequest)
		}

		request.InstallContainer.Args = append(request.InstallContainer.Args, args...)

		return true, nil
	}

	return false, nil
}

func isInjected(container corev1.Container, request *dtwebhook.BaseRequest) bool {
	if request.IsSplitMountsEnabled() {
		if (request.DynaKube.OneAgent().IsAppInjectionNeeded() && !volumes.HasSplitOneAgentMounts(&container)) ||
			(request.DynaKube.MetadataEnrichment().IsEnabled() && !volumes.HasSplitEnrichmentMounts(&container)) {
			return false
		}

		return true
	} else {
		return volumes.HasCommonConfigVolumeMounts(&container)
	}
}
