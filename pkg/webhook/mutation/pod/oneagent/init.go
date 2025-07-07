package oneagent

import (
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/move"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/arg"
	corev1 "k8s.io/api/core/v1"
)

func mutateInitContainer(mutationRequest *dtwebhook.MutationRequest, installPath string) error {
	isCSI := IsCSIVolume(mutationRequest.BaseRequest)
	isSelfExtractingImage := IsSelfExtractingImage(mutationRequest.BaseRequest, isCSI)

	if isCSI {
		log.Info("configuring init-container with CSI bin volume", "name", mutationRequest.PodName())
		addCSIBinVolume(
			mutationRequest.Pod,
			mutationRequest.DynaKube.Name,
			mutationRequest.DynaKube.FF().GetCSIMaxRetryTimeout().String())
	} else {
		log.Info("configuring init-container with emptyDir bin volume", "name", mutationRequest.PodName())
		addEmptyDirBinVolume(mutationRequest.Pod)
	}

	if isSelfExtractingImage {
		log.Info("configuring init-container with self-extracting image", "name", mutationRequest.PodName())
		// The first element would be the "bootstrap" sub command, which is not needed incase of self-extracting image
		mutationRequest.InstallContainer.Args = mutationRequest.InstallContainer.Args[1:]

	} else if !isCSI {
		log.Info("configuring init-container for ZIP download", "name", mutationRequest.PodName())
		downloadArgs := []arg.Arg{
			{Name: bootstrapper.TargetVersionFlag, Value: mutationRequest.DynaKube.OneAgent().GetCodeModulesVersion()},
		}

		if flavor := maputils.GetField(mutationRequest.Pod.Annotations, AnnotationFlavor, ""); flavor != "" {
			downloadArgs = append(downloadArgs,
				arg.Arg{Name: bootstrapper.FlavorFlag, Value: flavor})
		}

		mutationRequest.InstallContainer.Args = append(mutationRequest.InstallContainer.Args, arg.ConvertArgsToStrings(downloadArgs)...)
	}

	addInitVolumeMounts(mutationRequest.InstallContainer)

	return addInitArgs(*mutationRequest.Pod, mutationRequest.InstallContainer, mutationRequest.DynaKube, installPath)
}

func addInitArgs(pod corev1.Pod, initContainer *corev1.Container, dk dynakube.DynaKube, installPath string) error {
	args := []arg.Arg{
		{Name: cmd.SourceFolderFlag, Value: consts.AgentCodeModuleSource},
		{Name: cmd.TargetFolderFlag, Value: binInitMountPath},
		{Name: configure.InstallPathFlag, Value: installPath},
	}

	if dk.OneAgent().IsCloudNativeFullstackMode() {
		log.Info("configuring init-container to setup fullstack mode", "pod.name", pod.GetName(), "pod.generateName", pod.GetGenerateName())
		tenantUUID, err := dk.TenantUUID()
		if err != nil {
			return err
		}

		args = append(args, arg.Arg{Name: configure.IsFullstackFlag}, arg.Arg{Name: configure.TenantFlag, Value: tenantUUID})
	}

	if technology := getTechnology(pod, dk); technology != "" {
		args = append(args, arg.Arg{Name: move.TechnologyFlag, Value: technology})
	}

	if initContainer.Args == nil {
		initContainer.Args = []string{}
	}

	initContainer.Args = append(initContainer.Args, arg.ConvertArgsToStrings(args)...)

	return nil
}

func getTechnology(pod corev1.Pod, dk dynakube.DynaKube) string {
	if technology, ok := pod.Annotations[AnnotationTechnologies]; ok {
		return technology
	}

	technology := dk.FF().GetNodeImagePullTechnology()
	if technology != "" {
		return technology
	}

	return ""
}

func HasPodUserSet(ctx *corev1.PodSecurityContext) bool {
	return ctx != nil && ctx.RunAsUser != nil
}

func HasPodGroupSet(ctx *corev1.PodSecurityContext) bool {
	return ctx != nil && ctx.RunAsGroup != nil
}

func IsNonRoot(ctx *corev1.SecurityContext) bool {
	return ctx != nil &&
		(ctx.RunAsUser != nil && *ctx.RunAsUser != RootUserGroup) &&
		(ctx.RunAsGroup != nil && *ctx.RunAsGroup != RootUserGroup)
}
