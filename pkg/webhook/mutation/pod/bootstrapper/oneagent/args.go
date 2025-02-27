package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/arg"
	corev1 "k8s.io/api/core/v1"
)

func addInitArgs(pod corev1.Pod, initContainer *corev1.Container, dk dynakube.DynaKube) {
	args := []arg.Arg{
		{Name: bootstrapperSourceArgument, Value: consts.AgentCodeModuleSource},
		{Name: bootstrapperTargetArgument, Value: consts.AgentBinDirMount},
	}

	if technology := getTechnology(pod, dk); technology != "" {
		args = append(args, arg.Arg{Name: bootstrapperTechnologyArgument, Value: technology})
	}

	initContainer.Args = arg.ConvertArgsToStrings(args)
}

func getTechnology(pod corev1.Pod, dk dynakube.DynaKube) string {
	if technology, ok := pod.Annotations[dynakube.AnnotationFeatureRemoteDownloadTechnology]; ok {
		return technology
	}

	if dk.FeatureRemoteDownloadTechnology() != "" {
		return dk.FeatureRemoteDownloadTechnology()
	}

	return ""
}
