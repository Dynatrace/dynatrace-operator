package v2

import (
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func createInitContainerBase(pod *corev1.Pod, dk dynakube.DynaKube) (*corev1.Container, error) {
	customImage := dk.OneAgent().GetCustomCodeModulesImage()
	if customImage == "" {
		return nil, errors.New("custom code modules image not set")
	}

	initContainer := &corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           customImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: securityContextForInitContainer(pod, dk),
		Resources:       initContainerResources(dk),
	}

	return initContainer, nil
}

func addInitContainerToPod(pod *corev1.Pod, initContainer *corev1.Container) {
	common.AddInitConfigVolumeMount(initContainer)
	common.AddInitInputVolumeMount(initContainer)
	// TODO: Add all `--attribute` args to init-container
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *initContainer)
	common.AddInputVolume(pod)
	common.AddConfigVolume(pod)
}

func initContainerResources(dk dynakube.DynaKube) corev1.ResourceRequirements {
	customInitResources := dk.OneAgent().GetInitResources()
	if customInitResources != nil {
		return *customInitResources
	}

	return corev1.ResourceRequirements{}
}

func securityContextForInitContainer(pod *corev1.Pod, dk dynakube.DynaKube) *corev1.SecurityContext {
	initSecurityCtx := corev1.SecurityContext{
		ReadOnlyRootFilesystem:   ptr.To(true),
		AllowPrivilegeEscalation: ptr.To(false),
		Privileged:               ptr.To(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		RunAsUser:  ptr.To(oacommon.DefaultUser),
		RunAsGroup: ptr.To(oacommon.DefaultGroup),
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
	if dk.FeatureInitContainerSeccomp() {
		ctx.SeccompProfile = &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}
	}
}
