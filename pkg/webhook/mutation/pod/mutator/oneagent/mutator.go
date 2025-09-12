package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/mounts"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	CSIVolumeType       = "csi"
	EphemeralVolumeType = "ephemeral"
)

type Mutator struct{}

func NewMutator() dtwebhook.Mutator {
	return &Mutator{}
}

func IsSelfExtractingImage(mutationRequest *dtwebhook.BaseRequest) bool {
	ffEnabled := mutationRequest.DynaKube.FF().IsNodeImagePull()

	return ffEnabled && !isCSIVolume(mutationRequest)
}

func isCSIVolume(mutationRequest *dtwebhook.BaseRequest) bool {
	defaultVolumeType := EphemeralVolumeType
	if mutationRequest.DynaKube.OneAgent().IsCSIAvailable() {
		defaultVolumeType = CSIVolumeType
	}

	if mutationRequest.DynaKube.FF().IsNodeImagePull() {
		return maputils.GetField(mutationRequest.Pod.Annotations, AnnotationVolumeType, defaultVolumeType) == CSIVolumeType
	}

	return defaultVolumeType == CSIVolumeType
}

func IsEnabled(request *dtwebhook.BaseRequest) bool {
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, AnnotationInject, request.DynaKube.FF().IsAutomaticInjection())
	enabledOnDynakube := request.DynaKube.OneAgent().GetNamespaceSelector() != nil

	matchesNamespaceSelector := true // if no namespace selector is configured, we just pass set this to true

	if request.DynaKube.OneAgent().GetNamespaceSelector().Size() > 0 {
		selector, _ := metav1.LabelSelectorAsSelector(request.DynaKube.OneAgent().GetNamespaceSelector())

		matchesNamespaceSelector = selector.Matches(labels.Set(request.Namespace.Labels))
	}

	return matchesNamespaceSelector && enabledOnPod && enabledOnDynakube
}

func (mut *Mutator) IsEnabled(request *dtwebhook.BaseRequest) bool {
	return IsEnabled(request)
}

func (mut *Mutator) IsInjected(request *dtwebhook.BaseRequest) bool {
	return maputils.GetFieldBool(request.Pod.Annotations, AnnotationInjected, false)
}

func (mut *Mutator) Mutate(request *dtwebhook.MutationRequest) error {
	installPath := maputils.GetField(request.Pod.Annotations, AnnotationInstallPath, DefaultInstallPath)

	err := mutateInitContainer(request, installPath)
	if err != nil {
		return err
	}

	// not checking the returned bool, as getting a `false` value shouldn't happen
	// the caller of mutate already checks if it needs to be mutated
	_ = mutateUserContainers(request.BaseRequest, installPath)
	setInjectedAnnotation(request.Pod)

	return nil
}

func (mut *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	installPath := maputils.GetField(request.Pod.Annotations, AnnotationInstallPath, DefaultInstallPath)

	return mutateUserContainers(request.BaseRequest, installPath)
}

func containerIsInjected(container corev1.Container) bool {
	return mounts.IsIn(container.VolumeMounts, BinVolumeName)
}

func mutateUserContainers(request *dtwebhook.BaseRequest, installPath string) bool {
	newContainers := request.NewContainers(containerIsInjected)
	for _, container := range newContainers {
		addOneAgentToContainer(request.DynaKube, container, request.Namespace, installPath)
	}

	return len(newContainers) > 0
}

func addOneAgentToContainer(dk dynakube.DynaKube, container *corev1.Container, namespace corev1.Namespace, installPath string) {
	log.Info("adding OneAgent to container", "name", container.Name)

	addVolumeMounts(container, installPath)
	addDeploymentMetadataEnv(container, dk)
	addPreloadEnv(container, installPath)
	addDtStorageEnv(container)

	if dk.Spec.NetworkZone != "" {
		addNetworkZoneEnv(container, dk.Spec.NetworkZone)
	}

	if dk.FF().IsLabelVersionDetection() {
		addVersionDetectionEnvs(container, namespace)
	}
}

func setInjectedAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[AnnotationInjected] = "true"
	delete(pod.Annotations, AnnotationReason)
}

func setNotInjectedAnnotationFunc(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[AnnotationInjected] = "false"
		pod.Annotations[AnnotationReason] = reason
	}
}
