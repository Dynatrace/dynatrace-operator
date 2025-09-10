package pod

import (
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/arg"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func (wh *dtInjectionWebhook) createInitContainerBase(pod *corev1.Pod, dk dynakube.DynaKube) *corev1.Container {
	args := []arg.Arg{
		{
			Name:  configure.ConfigFolderFlag,
			Value: volumes.InitConfigMountPath,
		},
		{
			Name:  configure.InputFolderFlag,
			Value: volumes.InitInputMountPath,
		},
	}

	if areErrorsSuppressed(pod, dk) {
		args = append(args, arg.Arg{Name: cmd.SuppressErrorsFlag})
	}

	initContainer := &corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           wh.webhookPodImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: securityContextForInitContainer(pod, dk, wh.isOpenShift),
		Resources:       initContainerResources(dk),
		Args:            []string{bootstrapper.Use},
	}

	initContainer.Args = append(initContainer.Args, arg.ConvertArgsToStrings(args)...)

	return initContainer
}

func areErrorsSuppressed(pod *corev1.Pod, dk dynakube.DynaKube) bool {
	return maputils.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, dk.FF().GetInjectionFailurePolicy()) != "fail" // safer than == silent
}

func addInitContainerToPod(pod *corev1.Pod, initContainer *corev1.Container) {
	volumes.AddInitConfigVolumeMount(initContainer)
	volumes.AddInitInputVolumeMount(initContainer)
	volumes.AddInputVolume(pod)
	volumes.AddConfigVolume(pod)
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *initContainer)
}

func initContainerResources(dk dynakube.DynaKube) corev1.ResourceRequirements {
	customInitResources := dk.OneAgent().GetInitResources()
	if customInitResources != nil {
		return *customInitResources
	}

	return corev1.ResourceRequirements{}
}

func securityContextForInitContainer(pod *corev1.Pod, dk dynakube.DynaKube, isOpenShift bool) *corev1.SecurityContext {
	initSecurityCtx := corev1.SecurityContext{
		ReadOnlyRootFilesystem:   ptr.To(true),
		AllowPrivilegeEscalation: ptr.To(false),
		Privileged:               ptr.To(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		RunAsGroup: ptr.To(oacommon.DefaultGroup),
	}

	if !isOpenShift {
		initSecurityCtx.RunAsUser = ptr.To(oacommon.DefaultUser)
	}

	addSeccompProfile(&initSecurityCtx, dk)

	return combineSecurityContexts(initSecurityCtx, *pod)
}

func combineSecurityContexts(baseSecurityCtx corev1.SecurityContext, pod corev1.Pod) *corev1.SecurityContext {
	podSecurityCtx := pod.Spec.SecurityContext

	if oacommon.HasPodUserSet(podSecurityCtx) {
		baseSecurityCtx.RunAsUser = podSecurityCtx.RunAsUser
	}

	if oacommon.HasPodGroupSet(podSecurityCtx) {
		baseSecurityCtx.RunAsGroup = podSecurityCtx.RunAsGroup
	}

	baseSecurityCtx.RunAsNonRoot = ptr.To(oacommon.IsNonRoot(&baseSecurityCtx))

	return &baseSecurityCtx
}

func addSeccompProfile(ctx *corev1.SecurityContext, dk dynakube.DynaKube) {
	if dk.FF().HasInitSeccomp() {
		ctx.SeccompProfile = &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}
	}
}
