package injection

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8smount"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
)

func addPodAttributes(request *mutator.MutationRequest) error {

	attrs := podattr.Attributes{}
	attrs, envs := attributes.GetPodAttributes(attrs, request.BaseRequest)

	request.InstallContainer.Env = append(request.InstallContainer.Env, envs...)

	args, err := podattr.ToArgs(attrs)
	if err != nil {
		return err
	}

	request.InstallContainer.Args = append(request.InstallContainer.Args, args...)

	return nil
}

func addContainerAttributes(request *mutator.MutationRequest) (bool, error) {
	newContainers := request.NewContainers(isInjected)
	attributes := attributes.GetContainerAttributes(request, newContainers)

	for _, c := range newContainers {
		volumes.AddConfigVolumeMount(c, request.BaseRequest)
	}

	if len(attributes) > 0 {
		args, err := containerattr.ToArgs(attributes)
		if err != nil {
			return false, err
		}

		request.InstallContainer.Args = append(request.InstallContainer.Args, args...)

		return true, nil
	}

	return false, nil
}

func isInjected(container corev1.Container, request *mutator.BaseRequest) bool {
	if request.IsSplitMountsEnabled() {
		if request.DynaKube.OneAgent().IsAppInjectionNeeded() && !k8smount.ContainsPath(container.VolumeMounts, volumes.ConfigMountPathOneAgent) ||
			request.DynaKube.MetadataEnrichment().IsEnabled() && !k8smount.ContainsPath(container.VolumeMounts, volumes.ConfigMountPathEnrichment) {
			return false
		}

		return true
	} else {
		return k8smount.ContainsPath(container.VolumeMounts, volumes.ConfigMountPath)
	}
}
