// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8smount"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CSIVolumeType       = "csi"
	EphemeralVolumeType = "ephemeral"
)

type invalidInstallPathError struct {
	InstallPath string
}

func (err invalidInstallPathError) Error() string {
	return fmt.Sprintf("the installPath (%s) must be clean, absolute and without whitespace and separators like ,:", err.InstallPath)
}

type Mutator struct {
	client client.Client
}

func NewMutator(clt client.Client) dtwebhook.Mutator {
	return &Mutator{client: clt}
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
	_, log := logd.NewFromContext(request.Context, "oneagent")
	installPath := maputils.GetField(request.Pod.Annotations, AnnotationInstallPath, DefaultInstallPath)

	if err := validateInstallPath(installPath); err != nil {
		return err
	}

	err := mutateInitContainer(request, installPath)
	if err != nil {
		return err
	}

	runtimeClassHandler := mut.resolveRuntimeClassHandler(request.Context, request.Pod.Spec.RuntimeClassName, log)

	// not checking the returned bool, as getting a `false` value shouldn't happen
	// the caller of mutate already checks if it needs to be mutated
	_ = mutateUserContainers(request.BaseRequest, installPath, runtimeClassHandler, log)
	setInjectedAnnotation(request.Pod)

	return nil
}

func (mut *Mutator) Reinvoke(ctx context.Context, request *dtwebhook.ReinvocationRequest) bool {
	_, log := logd.NewFromContext(ctx, "oneagent")

	installPath := maputils.GetField(request.Pod.Annotations, AnnotationInstallPath, DefaultInstallPath)
	if err := validateInstallPath(installPath); err != nil {
		return false
	}

	runtimeClassHandler := mut.resolveRuntimeClassHandler(ctx, request.Pod.Spec.RuntimeClassName, log)

	return mutateUserContainers(request.BaseRequest, installPath, runtimeClassHandler, log)
}

func (mut *Mutator) resolveRuntimeClassHandler(ctx context.Context, runtimeClassName *string, log logd.Logger) string {
	if runtimeClassName == nil {
		return ""
	}

	var runtimeClass nodev1.RuntimeClass
	if err := mut.client.Get(ctx, client.ObjectKey{Name: *runtimeClassName}, &runtimeClass); err != nil {
		log.Error(err, "failed to get RuntimeClass, skipping env injection", "name", *runtimeClassName)

		return ""
	}

	return runtimeClass.Handler
}

func containerIsInjected(container corev1.Container, _ *dtwebhook.BaseRequest) bool {
	return k8smount.Contains(container.VolumeMounts, BinVolumeName)
}

func mutateUserContainers(request *dtwebhook.BaseRequest, installPath string, runtimeClassHandler string, log logd.Logger) bool {
	newContainers := request.NewContainers(containerIsInjected)

	for _, container := range newContainers {
		log.Info("adding OneAgent to container", "name", container.Name)

		addVolumeMounts(container, installPath)
		addDeploymentMetadataEnv(container, request.DynaKube)
		addPreloadEnv(container, installPath)
		addDTStorageEnv(container)

		if request.DynaKube.Spec.NetworkZone != "" {
			addNetworkZoneEnv(container, request.DynaKube.Spec.NetworkZone)
		}

		if request.DynaKube.FF().IsLabelVersionDetection() {
			addVersionDetectionEnvs(container, request.Namespace)
		}

		if runtimeClassHandler != "" {
			addPodRuntimeClassEnv(container, runtimeClassHandler)
		}
	}

	return len(newContainers) > 0
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
