package pod_mutator

import (
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/resources"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
)

func createInstallInitContainerBase(webhookImage string, pod *corev1.Pod, dynakube dynatracev1beta1.DynaKube) *corev1.Container {
	return &corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           webhookImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"init"},
		Env: []corev1.EnvVar{
			{Name: consts.InjectionFailurePolicyEnv, Value: maputils.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, dynakube.FeatureInjectionFailurePolicy())},
			{Name: consts.K8sPodNameEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.name")},
			{Name: consts.K8sPodUIDEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.uid")},
			{Name: consts.K8sBasePodNameEnv, Value: getBasePodName(pod)},
			{Name: consts.K8sNamespaceEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.namespace")},
			{Name: consts.K8sNodeNameEnv, ValueFrom: env.NewEnvVarSourceForField("spec.nodeName")},
		},
		SecurityContext: securityContextForInitContainer(pod, dynakube),
		Resources:       initContainerResources(dynakube),
	}
}

func initContainerResources(dynakube dynatracev1beta1.DynaKube) corev1.ResourceRequirements {
	customInitResources := dynakube.InitResources()
	if customInitResources != nil {
		return *customInitResources
	}
	if !dynakube.NeedsCSIDriver() {
		return corev1.ResourceRequirements{}
	}
	return defaultInitContainerResources()
}

func defaultInitContainerResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: resources.NewResourceList("30m", "30Mi"),
		Limits:   resources.NewResourceList("100m", "60Mi"),
	}
}

func securityContextForInitContainer(pod *corev1.Pod, dk dynatracev1beta1.DynaKube) *corev1.SecurityContext {
	initSecurityCtx := corev1.SecurityContext{
		ReadOnlyRootFilesystem:   address.Of(true),
		AllowPrivilegeEscalation: address.Of(false),
		Privileged:               address.Of(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}

	addSeccompProfile(&initSecurityCtx, dk)

	return combineSecurityContexts(initSecurityCtx, *pod)
}

// combineSecurityContexts returns a SecurityContext that combines the provided SecurityContext
// with the user/group of the provided Pod's SecurityContext and the 1. container's SecurityContext
func combineSecurityContexts(baseSecurityCtx corev1.SecurityContext, pod corev1.Pod) *corev1.SecurityContext {
	containerSecurityCtx := pod.Spec.Containers[0].SecurityContext
	podSecurityCtx := pod.Spec.SecurityContext

	baseSecurityCtx.RunAsUser = address.Of(defaultUser)
	baseSecurityCtx.RunAsGroup = address.Of(defaultGroup)

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

	baseSecurityCtx.RunAsNonRoot = address.Of(isNonRoot(&baseSecurityCtx))

	return &baseSecurityCtx
}

func hasPodUserSet(ctx *corev1.PodSecurityContext) bool {
	return ctx != nil && ctx.RunAsUser != nil
}

func hasPodGroupSet(ctx *corev1.PodSecurityContext) bool {
	return ctx != nil && ctx.RunAsGroup != nil
}

func hasContainerUserSet(ctx *corev1.SecurityContext) bool {
	return ctx != nil && ctx.RunAsUser != nil
}

func hasContainerGroupSet(ctx *corev1.SecurityContext) bool {
	return ctx != nil && ctx.RunAsGroup != nil
}

func isNonRoot(ctx *corev1.SecurityContext) bool {
	return ctx != nil &&
		(ctx.RunAsUser != nil && *ctx.RunAsUser != rootUserGroup) &&
		(ctx.RunAsGroup != nil && *ctx.RunAsGroup != rootUserGroup)
}

func getBasePodName(pod *corev1.Pod) string {
	basePodName := k8spod.GetName(*pod)

	// Only include up to the last dash character, exclusive.
	if lastDashIndex := strings.LastIndex(basePodName, "-"); lastDashIndex != -1 {
		basePodName = basePodName[:lastDashIndex]
	}
	return basePodName
}

func addInitContainerToPod(pod *corev1.Pod, initContainer *corev1.Container) {
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *initContainer)
}

func addSeccompProfile(ctx *corev1.SecurityContext, dk dynatracev1beta1.DynaKube) {
	if dk.FeatureInitContainerSeccomp() {
		ctx.SeccompProfile = &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}
	}
}
