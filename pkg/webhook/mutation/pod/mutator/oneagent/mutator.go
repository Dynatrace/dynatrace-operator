package oneagent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8smount"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	CSIVolumeType       = "csi"
	EphemeralVolumeType = "ephemeral"

	noExemption = ""
)

type invalidInstallPathError struct {
	InstallPath string
}

func (err invalidInstallPathError) Error() string {
	return fmt.Sprintf("the installPath (%s) must be clean, absolute and without whitespace and separators like ,:", err.InstallPath)
}

type bootstrapperSecretVolumeError struct {
	VolumeName string
	SecretName string
}

func (err bootstrapperSecretVolumeError) Error() string {
	return fmt.Sprintf("volume %q is based on the reserved bootstrapper secret %q", err.VolumeName, err.SecretName)
}

type bootstrapperSecretVolumeMountError struct {
	ContainerName string
}

func (err bootstrapperSecretVolumeMountError) Error() string {
	return fmt.Sprintf("container %q mounts the reserved volume %q", err.ContainerName, volumes.InputVolumeName)
}

type Mutator struct{}

func NewMutator() dtwebhook.Mutator {
	return &Mutator{}
}

func IsSelfExtractingImage(mutationRequest *dtwebhook.BaseRequest) bool {
	hasImage := mutationRequest.DynaKube.OneAgent().GetCodeModulesImage() != ""

	return hasImage && !isCSIVolume(mutationRequest)
}

func isCSIVolume(mutationRequest *dtwebhook.BaseRequest) bool {
	defaultVolumeType := EphemeralVolumeType
	if mutationRequest.DynaKube.OneAgent().IsCSIAvailable() {
		defaultVolumeType = CSIVolumeType
	}

	if mutationRequest.DynaKube.OneAgent().GetCodeModulesImage() != "" {
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

func (mut *Mutator) IsEnabled(_ context.Context, request *dtwebhook.BaseRequest) bool {
	return IsEnabled(request)
}

func (mut *Mutator) IsInjected(_ context.Context, request *dtwebhook.BaseRequest) bool {
	return maputils.GetFieldBool(request.Pod.Annotations, AnnotationInjected, false)
}

func validateBootstrapperSecretVolumeMounts(log logd.Logger, pod *corev1.Pod, skipDynatraceOperatorInitContainerByName string) error {
	validate := func(container corev1.Container) error {
		for _, mount := range container.VolumeMounts {
			if mount.Name == volumes.InputVolumeName {
				return dtwebhook.MutatorError{
					Err:      bootstrapperSecretVolumeMountError{ContainerName: container.Name},
					Annotate: setNotInjectedAnnotationFunc(BootstrapperSecretMountedReason),
				}
			}
		}

		return nil
	}

	for _, container := range pod.Spec.InitContainers {
		if container.Name == skipDynatraceOperatorInitContainerByName {
			continue
		}

		if err := validate(container); err != nil {
			log.Info("init container mounts the reserved input volume, injection skipped",
				"container", container.Name, "volume", volumes.InputVolumeName)

			return err
		}
	}

	for _, container := range pod.Spec.Containers {
		if err := validate(container); err != nil {
			log.Info("container mounts the reserved input volume, injection skipped",
				"container", container.Name, "volume", volumes.InputVolumeName)

			return err
		}
	}

	return nil
}

func validateBootstrapperSecretVolumes(log logd.Logger, pod *corev1.Pod, skipDynatraceInputVolumeByName string) error {
	for _, v := range pod.Spec.Volumes {
		if v.Name == skipDynatraceInputVolumeByName {
			continue
		}

		switch {
		case v.Secret != nil && isBootstrapperSecret(v.Secret.SecretName):
			err := dtwebhook.MutatorError{
				Err:      bootstrapperSecretVolumeError{VolumeName: v.Name, SecretName: v.Secret.SecretName},
				Annotate: setNotInjectedAnnotationFunc(BootstrapperSecretMountedReason),
			}
			log.Info("volume references a reserved bootstrapper secret, injection skipped",
				"volume", v.Name, "secret", v.Secret.SecretName)

			return err
		case v.Projected != nil:
			for _, source := range v.Projected.Sources {
				if source.Secret != nil && isBootstrapperSecret(source.Secret.Name) {
					err := dtwebhook.MutatorError{
						Err:      bootstrapperSecretVolumeError{VolumeName: v.Name, SecretName: source.Secret.Name},
						Annotate: setNotInjectedAnnotationFunc(BootstrapperSecretMountedReason),
					}
					log.Info("projected volume sources a reserved bootstrapper secret, injection skipped",
						"volume", v.Name, "secret", source.Secret.Name)

					return err
				}
			}
		}
	}

	return nil
}

func isBootstrapperSecret(name string) bool {
	return name == consts.BootstrapperInitSecretName || name == consts.BootstrapperInitCertsSecretName
}

func validateInstallPath(installPath string) error {
	if !filepath.IsAbs(installPath) ||
		installPath == string(os.PathSeparator) ||
		strings.ContainsFunc(installPath, unicode.IsSpace) ||
		strings.ContainsAny(installPath, "\x00,:") ||
		filepath.Clean(installPath) != installPath {
		return dtwebhook.MutatorError{
			Err:      invalidInstallPathError{installPath},
			Annotate: setNotInjectedAnnotationFunc(InvalidInstallPathReason),
		}
	}

	return nil
}

func (mut *Mutator) Mutate(request *dtwebhook.MutationRequest) error {
	_, log := logd.NewFromContext(request.Context, "oa-mutation")
	installPath := maputils.GetField(request.Pod.Annotations, AnnotationInstallPath, DefaultInstallPath)

	if err := validateInstallPath(installPath); err != nil {
		return err
	}

	if err := validateBootstrapperSecretVolumeMounts(log, request.Pod, noExemption); err != nil {
		return err
	}

	if err := validateBootstrapperSecretVolumes(log, request.Pod, noExemption); err != nil {
		return err
	}

	err := mutateInitContainer(request, installPath)
	if err != nil {
		return err
	}

	// not checking the returned bool, as getting a `false` value shouldn't happen
	// the caller of mutate already checks if it needs to be mutated
	_ = mutateUserContainers(request.BaseRequest, installPath, log)
	setInjectedAnnotation(request.Pod)

	return nil
}

func (mut *Mutator) Reinvoke(ctx context.Context, request *dtwebhook.ReinvocationRequest) bool {
	_, log := logd.NewFromContext(ctx, "oa-mutation")

	installPath := maputils.GetField(request.Pod.Annotations, AnnotationInstallPath, DefaultInstallPath)
	if err := validateInstallPath(installPath); err != nil {
		return false
	}

	if err := validateBootstrapperSecretVolumeMounts(log, request.Pod, dtwebhook.InstallContainerName); err != nil {
		return false
	}

	if err := validateBootstrapperSecretVolumes(log, request.Pod, volumes.InputVolumeName); err != nil {
		return false
	}

	return mutateUserContainers(request.BaseRequest, installPath, log)
}

func containerIsInjected(container corev1.Container, _ *dtwebhook.BaseRequest) bool {
	return k8smount.Contains(container.VolumeMounts, BinVolumeName)
}

func mutateUserContainers(request *dtwebhook.BaseRequest, installPath string, log logd.Logger) bool {
	newContainers := request.NewContainers(containerIsInjected)
	for _, container := range newContainers {
		addOneAgentToContainer(request.DynaKube, container, request.Namespace, installPath, log)
	}

	return len(newContainers) > 0
}

func addOneAgentToContainer(dk dynakube.DynaKube, container *corev1.Container, namespace corev1.Namespace, installPath string, log logd.Logger) {
	log.Info("adding OneAgent to container", "name", container.Name)

	addVolumeMounts(container, installPath)
	addDeploymentMetadataEnv(container, dk)
	addPreloadEnv(container, installPath)
	addDTStorageEnv(container)

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
