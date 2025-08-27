package oneagent

import (
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/move"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/arg"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
)

func mutateInitContainer(mutationRequest *dtwebhook.MutationRequest, installPath string) error {
	if isCSIVolume(mutationRequest.BaseRequest) {
		log.Info("configuring init-container with CSI bin volume", "name", mutationRequest.PodName())
		addCSIBinVolume(
			mutationRequest.Pod,
			mutationRequest.DynaKube.Name,
			mutationRequest.DynaKube.FF().GetCSIMaxRetryTimeout().String())
		// in case of CSI, the CSI volume itself is already always readonly, so the mount should always be readonly, the init-container should just read from it
		addInitBinMount(mutationRequest.InstallContainer, true)
	} else {
		log.Info("configuring init-container with emptyDir bin volume", "name", mutationRequest.PodName())
		addEmptyDirBinVolume(mutationRequest.Pod)
		// in case of no CSI, the the emptyDir can't be readonly for the init-container, as it first has to download/move the agent into it
		addInitBinMount(mutationRequest.InstallContainer, false)

		if mutationRequest.DynaKube.FF().IsNodeImagePull() {
			log.Info("configuring init-container with self-extracting image", "name", mutationRequest.PodName())
			// The first element would be the "bootstrap" subcommand, which is not needed in case of self-extracting image
			mutationRequest.InstallContainer.Args = mutationRequest.InstallContainer.Args[1:]
			mutationRequest.InstallContainer.Image = mutationRequest.DynaKube.OneAgent().GetCodeModulesImage()
		} else {
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
	}

	return addInitArgs(*mutationRequest.Pod, mutationRequest.InstallContainer, mutationRequest.DynaKube, installPath)
}

func addInitArgs(pod corev1.Pod, initContainer *corev1.Container, dk dynakube.DynaKube, installPath string) error {
	args := []arg.Arg{
		{Name: cmd.SourceFolderFlag, Value: AgentCodeModuleSource},
		{Name: cmd.TargetFolderFlag, Value: consts.AgentInitBinDirMount},
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
	return maputils.GetField(pod.Annotations, AnnotationTechnologies, dk.FF().GetNodeImagePullTechnology())
}

func HasPodUserSet(psc *corev1.PodSecurityContext) bool {
	return psc != nil && psc.RunAsUser != nil
}

func HasPodGroupSet(psc *corev1.PodSecurityContext) bool {
	return psc != nil && psc.RunAsGroup != nil
}

func IsNonRoot(sc *corev1.SecurityContext) bool {
	if sc == nil {
		return true
	}

	if sc.RunAsUser != nil && *sc.RunAsUser != RootUser {
		return true
	}

	if sc.RunAsGroup != nil && *sc.RunAsGroup != RootGroup {
		return true
	}

	return false
}
