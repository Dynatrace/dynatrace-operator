package pod_mutator

import (
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	corev1 "k8s.io/api/core/v1"
)

func createInstallInitContainerBase(webhookImage, clusterID string, pod *corev1.Pod, dynakube dynatracev1beta1.DynaKube) *corev1.Container {
	return &corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           webhookImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"init"},
		Env: []corev1.EnvVar{
			{Name: config.AgentContainerCountEnv, Value: strconv.Itoa(len(pod.Spec.Containers))},
			{Name: config.InjectionFailurePolicyEnv, Value: kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent")},
			{Name: config.K8sPodNameEnv, ValueFrom: kubeobjects.NewEnvVarSourceForField("metadata.name")},
			{Name: config.K8sPodUIDEnv, ValueFrom: kubeobjects.NewEnvVarSourceForField("metadata.uid")},
			{Name: config.K8sBasePodNameEnv, Value: getBasePodName(pod)},
			{Name: config.K8sClusterIDEnv, Value: clusterID},
			{Name: config.K8sNamespaceEnv, ValueFrom: kubeobjects.NewEnvVarSourceForField("metadata.namespace")},
			{Name: config.K8sNodeNameEnv, ValueFrom: kubeobjects.NewEnvVarSourceForField("spec.nodeName")},
		},
		SecurityContext: copyUserContainerSecurityContext(pod),
		Resources:       *dynakube.InitResources(),
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

func addInitContainerToPod(pod *corev1.Pod, initContainer *corev1.Container) {
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *initContainer)
}
