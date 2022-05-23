package pod

import (
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
)

func (webhook *podMutatorWebhook) createInstallInitContainerBase(pod *corev1.Pod, dynakube *dynatracev1beta1.DynaKube) *corev1.Container {
	return &corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           webhook.webhookImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"init"},
		Env: []corev1.EnvVar{
			{Name: standalone.ContainerCountEnv, Value: strconv.Itoa(len(pod.Spec.Containers))},
			{Name: standalone.CanFailEnv, Value: kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent")},
			{Name: standalone.K8PodNameEnv, ValueFrom: kubeobjects.FieldEnvVar("metadata.name")},
			{Name: standalone.K8PodUIDEnv, ValueFrom: kubeobjects.FieldEnvVar("metadata.uid")},
			{Name: standalone.K8BasePodNameEnv, Value: getBasePodName(pod)},
			{Name: standalone.K8NamespaceEnv, ValueFrom: kubeobjects.FieldEnvVar("metadata.namespace")},
			{Name: standalone.K8NodeNameEnv, ValueFrom: kubeobjects.FieldEnvVar("spec.nodeName")},
		},
		SecurityContext: getSecurityContext(pod),
		VolumeMounts: []corev1.VolumeMount{
			{Name: injectionConfigVolumeName, MountPath: standalone.ConfigDirMount},
		},
		Resources: *dynakube.InitResources(),
	}
}

func getSecurityContext(pod *corev1.Pod) *corev1.SecurityContext {
	var sc *corev1.SecurityContext
	if pod.Spec.Containers[0].SecurityContext != nil {
		sc = pod.Spec.Containers[0].SecurityContext.DeepCopy()
	}
	return sc
}

func getBasePodName(pod *corev1.Pod) string {
	basePodName := pod.GenerateName
	if basePodName == "" {
		basePodName = pod.Name
	}

	// Only include up to the last dash character, exclusive.
	if p := strings.LastIndex(basePodName, "-"); p != -1 {
		basePodName = basePodName[:p]
	}
	return basePodName
}

func addToInitContainers(pod *corev1.Pod, installContainer *corev1.Container) {
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *installContainer)
}
