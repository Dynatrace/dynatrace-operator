// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package injection

import (
	"cmp"
	"context"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sresource"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8ssecuritycontext"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/arg"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
)

func (h *Handler) createInitContainerBase(ctx context.Context, pod *corev1.Pod, dk *dynakube.DynaKube) *corev1.Container {
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

	if bootstrapperconfig.NeedsDownloadConfig(dk) {
		args = append(args, arg.Arg{
			Name:  bootstrapper.BaseURL,
			Value: dk.APIURL(),
		})
	}

	if areErrorsSuppressed(pod, dk) {
		args = append(args, arg.Arg{Name: k8sinit.SuppressErrorsFlag})
	}

	initContainer := &corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           h.webhookPodImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: securityContextForInitContainer(ctx, pod, dk, h.isOpenShift),
		Resources:       defaultInitContainerResources(),
		Args:            append([]string{bootstrapper.Use}, arg.ConvertArgsToStrings(args)...),
	}

	return initContainer
}

func areErrorsSuppressed(pod *corev1.Pod, dk *dynakube.DynaKube) bool {
	return maputils.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, dk.FF().GetInjectionFailurePolicy()) != "fail" // safer than == silent
}

func addInitContainerToPod(ctx context.Context, pod *corev1.Pod, initContainer *corev1.Container) error {
	volumes.AddInitConfigVolumeMount(initContainer)
	volumes.AddInitInputVolumeMount(initContainer)

	if err := volumes.AddInputVolume(pod); err != nil {
		return err
	}

	if err := volumes.AddConfigVolume(ctx, pod); err != nil {
		return err
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *initContainer)

	return nil
}

func defaultInitContainerResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: k8sresource.NewResourceList("30m", "30Mi"),
		Limits:   k8sresource.NewResourceList("100m", "60Mi"),
	}
}

func securityContextForInitContainer(ctx context.Context, pod *corev1.Pod, dk *dynakube.DynaKube, isOpenShift bool) *corev1.SecurityContext {
	initSecurityCtx := corev1.SecurityContext{
		ReadOnlyRootFilesystem:   new(true),
		AllowPrivilegeEscalation: new(false),
		Privileged:               new(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		RunAsGroup: new(oacommon.DefaultGroup),
	}

	if !isOpenShift {
		initSecurityCtx.RunAsUser = new(oacommon.DefaultUser)
	}

	addSeccompProfile(&initSecurityCtx, dk)

	return combineSecurityContexts(ctx, initSecurityCtx, pod)
}

func combineSecurityContexts(ctx context.Context, baseSecurityCtx corev1.SecurityContext, pod *corev1.Pod) *corev1.SecurityContext {
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

	user, err := k8ssecuritycontext.GetValidatedRunAsID(pod.Annotations[dtwebhook.AnnotationInitContainerRunAsUser])
	if err != nil {
		logd.FromContext(ctx).Error(err, "invalid init container runAsUser annotation, ignoring", "annotation", dtwebhook.AnnotationInitContainerRunAsUser)
	} else if user != nil {
		baseSecurityCtx.RunAsUser = user
	}

	group, err := k8ssecuritycontext.GetValidatedRunAsID(pod.Annotations[dtwebhook.AnnotationInitContainerRunAsGroup])
	if err != nil {
		logd.FromContext(ctx).Error(err, "invalid init container runAsGroup annotation, ignoring", "annotation", dtwebhook.AnnotationInitContainerRunAsGroup)
	} else if group != nil {
		baseSecurityCtx.RunAsGroup = group
	}

	// if neither user nor group is set, the init container defaults to running as non-root.
	// Kubernetes's default behavior for runAsNonRoot: nil == runAsNonRoot: false. (it will just not do the root check)
	baseSecurityCtx.RunAsNonRoot = cmp.Or(k8ssecuritycontext.IsNonRoot(baseSecurityCtx.RunAsUser, baseSecurityCtx.RunAsGroup), new(true))

	return &baseSecurityCtx
}

func addSeccompProfile(ctx *corev1.SecurityContext, dk *dynakube.DynaKube) {
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
