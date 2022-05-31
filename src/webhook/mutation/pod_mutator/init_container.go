package pod_mutator

import (
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
)

func createInstallInitContainerBase(webhookImage string, pod *corev1.Pod, dynakube *dynatracev1beta1.DynaKube) *corev1.Container {
	return &corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           webhookImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"init"},
		Env: []corev1.EnvVar{
			{Name: standalone.ContainerCountEnv, Value: strconv.Itoa(len(pod.Spec.Containers))},
			{Name: standalone.CanFailEnv, Value: kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent")},
			{Name: standalone.K8PodNameEnv, ValueFrom: kubeobjects.NewEnvVarSourceForField("metadata.name")},
			{Name: standalone.K8PodUIDEnv, ValueFrom: kubeobjects.NewEnvVarSourceForField("metadata.uid")},
			{Name: standalone.K8BasePodNameEnv, Value: getBasePodName(pod)},
			{Name: standalone.K8NamespaceEnv, ValueFrom: kubeobjects.NewEnvVarSourceForField("metadata.namespace")},
			{Name: standalone.K8NodeNameEnv, ValueFrom: kubeobjects.NewEnvVarSourceForField("spec.nodeName")},
		},
		SecurityContext: copyUserContainerSecurityContext(pod),
		VolumeMounts: []corev1.VolumeMount{
			{Name: injectionConfigVolumeName, MountPath: standalone.ConfigDirMount},
		},
		Resources: *dynakube.InitResources(),
	}
}

func copyUserContainerSecurityContext(pod *corev1.Pod) *corev1.SecurityContext {
	var securityContext *corev1.SecurityContext
	if len(pod.Spec.Containers) == 0 {
		return securityContext
	}
	if pod.Spec.Containers[0].SecurityContext != nil {
		securityContext = pod.Spec.Containers[0].SecurityContext.DeepCopy()
	}
	return securityContext
}

func getBasePodName(pod *corev1.Pod) string {
	basePodName := pod.GenerateName
	if basePodName == "" {
		basePodName = pod.Name
	}

	// Only include up to the last dash character, exclusive.
	if lastDashIndex := strings.LastIndex(basePodName, "-"); lastDashIndex != -1 {
		basePodName = basePodName[:lastDashIndex]
	}
	return basePodName
}

func addToInitContainers(pod *corev1.Pod, initContainer *corev1.Container) {
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *initContainer)
}
