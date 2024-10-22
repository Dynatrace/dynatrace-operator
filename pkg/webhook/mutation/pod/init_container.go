package pod

import (
	"encoding/json"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/startup"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/resources"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/metadata"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/oneagent"
	corev1 "k8s.io/api/core/v1"
)

func createInstallInitContainerBase(webhookImage, clusterID string, pod *corev1.Pod, dk dynakube.DynaKube) *corev1.Container {
	return &corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           webhookImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"init"},
		Env: []corev1.EnvVar{
			{Name: consts.InjectionFailurePolicyEnv, Value: maputils.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, dk.FeatureInjectionFailurePolicy())},
			{Name: consts.K8sPodNameEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.name")},
			{Name: consts.K8sPodUIDEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.uid")},
			{Name: consts.K8sBasePodNameEnv, Value: getBasePodName(pod)},
			{Name: consts.K8sClusterIDEnv, Value: clusterID},
			{Name: consts.K8sNamespaceEnv, ValueFrom: env.NewEnvVarSourceForField("metadata.namespace")},
			{Name: consts.K8sNodeNameEnv, ValueFrom: env.NewEnvVarSourceForField("spec.nodeName")},
		},
		SecurityContext: securityContextForInitContainer(pod, dk),
		Resources:       initContainerResources(dk),
	}
}

func initContainerResources(dk dynakube.DynaKube) corev1.ResourceRequirements {
	customInitResources := dk.InitResources()
	if customInitResources != nil {
		return *customInitResources
	}

	if !dk.NeedsCSIDriver() {
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

func securityContextForInitContainer(pod *corev1.Pod, dk dynakube.DynaKube) *corev1.SecurityContext {
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
// with the user/group of the provided Pod's SecurityContext and the 1. container's SecurityContext.
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

func addSeccompProfile(ctx *corev1.SecurityContext, dk dynakube.DynaKube) {
	if dk.FeatureInitContainerSeccomp() {
		ctx.SeccompProfile = &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}
	}
}

func updateContainerInfo(request *dtwebhook.BaseRequest, installContainer *corev1.Container) bool {
	pod := request.Pod
	if installContainer == nil {
		installContainer = findInstallContainer(pod.Spec.InitContainers)
		if installContainer == nil {
			return false
		}
	}

	newContainers := request.NewContainers(containerIsInjected)
	if len(newContainers) == 0 {
		return false
	}

	containersEnv := env.FindEnvVar(installContainer.Env, consts.ContainerInfoEnv)

	var containersEnvValue []startup.ContainerInfo //nolint:prealloc

	if containersEnv == nil {
		containersEnv = &corev1.EnvVar{
			Name: consts.ContainerInfoEnv,
		}
	} else {
		json.Unmarshal([]byte(containersEnv.Value), &containersEnvValue)
	}

	for _, container := range newContainers {
		log.Info("updating init container with new container", "name", container.Name, "image", container.Image)
		containerInfo := startup.ContainerInfo{
			Name:  container.Name,
			Image: container.Image,
		}
		containersEnvValue = append(containersEnvValue, containerInfo)
	}

	rawEnv, err := json.Marshal(containersEnvValue)
	if err != nil {
		log.Error(err, "failed to create container info env var")
	}

	containersEnv.Value = string(rawEnv)

	installContainer.Env = env.AddOrUpdate(installContainer.Env, *containersEnv)

	return true
}

func findInstallContainer(initContainers []corev1.Container) *corev1.Container {
	for i := range initContainers {
		container := &initContainers[i]
		if container.Name == dtwebhook.InstallContainerName {
			return container
		}
	}

	return nil
}

func containerIsInjected(container corev1.Container) bool {
	if metadata.ContainerIsInjected(container) || oamutation.ContainerIsInjected(container) {
		return true
	}

	return false
}
