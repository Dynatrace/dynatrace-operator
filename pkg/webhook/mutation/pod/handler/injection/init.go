package injection

import (
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sresource"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/arg"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func (h *Handler) createInitContainerBase(pod *corev1.Pod, dk dynakube.DynaKube) *corev1.Container {
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
		args = append(args, arg.Arg{Name: k8sinit.SuppressErrorsFlag})
	}

	initContainer := &corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           h.webhookPodImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: securityContextForInitContainer(pod, dk, h.isOpenShift),
		Resources:       defaultInitContainerResources(),
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

func defaultInitContainerResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: k8sresource.NewResourceList("30m", "30Mi"),
		Limits:   k8sresource.NewResourceList("100m", "60Mi"),
	}
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
	containerSecurityCtx := &corev1.SecurityContext{}
	if len(pod.Spec.Containers) > 0 {
		containerSecurityCtx = pod.Spec.Containers[0].SecurityContext
	}

	podSecurityCtx := pod.Spec.SecurityContext

	if hasPodUserSet(podSecurityCtx) {
		baseSecurityCtx.RunAsUser = podSecurityCtx.RunAsUser
	}

	if hasPodGroupSet(podSecurityCtx) {
		baseSecurityCtx.RunAsGroup = podSecurityCtx.RunAsGroup
	}

	if hasContainerUserSet(containerSecurityCtx) {
		baseSecurityCtx.RunAsUser = containerSecurityCtx.RunAsUser
	}

	if hasContainerGroupSet(containerSecurityCtx) {
		baseSecurityCtx.RunAsGroup = containerSecurityCtx.RunAsGroup
	}

	baseSecurityCtx.RunAsNonRoot = ptr.To(isNonRoot(&baseSecurityCtx))

	return &baseSecurityCtx
}

func addSeccompProfile(ctx *corev1.SecurityContext, dk dynakube.DynaKube) {
	if dk.FF().HasInitSeccomp() {
		ctx.SeccompProfile = &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}
	}
}

func hasContainerUserSet(ctx *corev1.SecurityContext) bool {
	return ctx != nil && ctx.RunAsUser != nil
}

func hasContainerGroupSet(ctx *corev1.SecurityContext) bool {
	return ctx != nil && ctx.RunAsGroup != nil
}

func hasPodUserSet(psc *corev1.PodSecurityContext) bool {
	return psc != nil && psc.RunAsUser != nil
}

func hasPodGroupSet(psc *corev1.PodSecurityContext) bool {
	return psc != nil && psc.RunAsGroup != nil
}

func isNonRoot(sc *corev1.SecurityContext) bool {
	if sc == nil {
		return true
	}

	if sc.RunAsUser != nil { // user takes precedence over group
		return *sc.RunAsUser != RootUser
	}

	return sc.RunAsGroup == nil || *sc.RunAsGroup != RootGroup // if group is root, but no user is set, we are still "running as root"
}
