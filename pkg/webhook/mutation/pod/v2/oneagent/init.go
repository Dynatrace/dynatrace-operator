package oneagent

import (
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/move"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/common/arg"
	corev1 "k8s.io/api/core/v1"
)

const (
	bootstrapperSourceArgument = "source" // TODO import consts from bootstrapper
	bootstrapperTargetArgument = "target"
)

func mutateInitContainer(mutationRequest *dtwebhook.MutationRequest, installPath string) {
	addInitArgs(*mutationRequest.Pod, mutationRequest.InstallContainer, mutationRequest.DynaKube, installPath)
	addInitVolumeMounts(mutationRequest.InstallContainer)
}

func addInitArgs(pod corev1.Pod, initContainer *corev1.Container, dk dynakube.DynaKube, installPath string) {
	args := []arg.Arg{
		{Name: bootstrapperSourceArgument, Value: consts.AgentCodeModuleSource}, // TODO import consts from bootstrapper
		{Name: bootstrapperTargetArgument, Value: binInitMountPath},
		{Name: configure.InstallPathFlag, Value: installPath},
	}

	if technology := getTechnology(pod, dk); technology != "" {
		args = append(args, arg.Arg{Name: move.TechnologyFlag, Value: technology})
	}

	if initContainer.Args == nil {
		initContainer.Args = []string{}
	}

	initContainer.Args = append(initContainer.Args, arg.ConvertArgsToStrings(args)...)
}

func getTechnology(pod corev1.Pod, dk dynakube.DynaKube) string {
	if technology, ok := pod.Annotations[oacommon.AnnotationTechnologies]; ok {
		return technology
	}

	technology := dk.FeatureRemoteImageDownloadTechnology()
	if technology != "" {
		return technology
	}

	return ""
}
