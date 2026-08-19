// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	k8sversion "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

const (
	errorInvalidActiveGateCapability = `The DynaKube's specification tries to use an invalid capability in ActiveGate section, invalid capability=%s.
Make sure you correctly specify the ActiveGate capabilities in your custom resource.
`

	errorActiveGateInvalidPVCConfiguration = ` DynaKube specifies a PVC for the ActiveGate while ephemeral volume is also enabled. These settings are mutually exclusive, please choose only one.`

	warningMissingActiveGateMemoryLimit = `ActiveGate specification missing memory limits. Can cause excess memory usage.`

	warningActiveGateRollingUpdateOldK8sVersion = `ActiveGate rollingUpdate setting requires Kubernetes version 1.35 or higher. The current cluster version is below 1.35, so the rollingUpdate setting will be ignored.`

	errorActiveGateConflictingVolumeName      = `The DynaKube's ActiveGate specification uses a volume name that conflicts with an internally managed volume: %s. Please use a different name.`
	errorActiveGateConflictingVolumeMountPath = `The DynaKube's ActiveGate specification uses a volume mount path that conflicts with an internally managed volume: %s. Please use a different path.`
	errorActiveGateDisallowedVolumeType       = `The DynaKube's ActiveGate specification uses a volume "%s" with a type that is not allowed by the OpenShift nonroot-v2 SCC.`
	errorActiveGateInvalidVolumes             = `The DynaKube's ActiveGate specification has invalid volume configuration: %s`

	minK8sMinorVersionForRollingUpdate = 35
)

func invalidActiveGateCapabilities(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	if dk.ActiveGate().IsEnabled() {
		capabilities := dk.Spec.ActiveGate.Capabilities
		for _, capability := range capabilities {
			if _, ok := activegate.CapabilityDisplayNames[capability]; !ok {
				log.Info("requested dynakube has invalid active gate capability")

				return fmt.Sprintf(errorInvalidActiveGateCapability, capability)
			}
		}
	}

	return ""
}

func missingActiveGateMemoryLimit(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.ActiveGate().IsEnabled() &&
		!memoryLimitSet(dk.Spec.ActiveGate.Resources) {
		return warningMissingActiveGateMemoryLimit
	}

	return ""
}

func memoryLimitSet(resources corev1.ResourceRequirements) bool {
	return resources.Limits != nil && resources.Limits.Memory() != nil
}

func activeGateMutuallyExclusivePVCSettings(dk *dynakube.DynaKube) bool {
	return ptr.Deref(dk.Spec.ActiveGate.UseEphemeralVolume, false) && dk.Spec.ActiveGate.VolumeClaimTemplate != nil
}

func mutuallyExclusiveActiveGatePVsettings(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	if activeGateMutuallyExclusivePVCSettings(dk) {
		log.Info("requested dynakube specifies mutually exclusive VolumeClaimTemplate settings for ActiveGate.")

		return errorActiveGateInvalidPVCConfiguration
	}

	return ""
}

func activeGateRollingUpdateWithOldK8sVersion(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.ActiveGate.RollingUpdate == nil {
		return ""
	}

	if k8sversion.GetMinorVersion() < minK8sMinorVersionForRollingUpdate {
		return warningActiveGateRollingUpdateOldK8sVersion
	}

	return ""
}

func activeGateHasConflictingVolumes(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	for _, volume := range dk.Spec.ActiveGate.Volumes {
		if slices.Contains(agconsts.VolumeNames, volume.Name) {
			log.Info("conflicting ActiveGate volume name detected", "volume", volume.Name)

			return fmt.Sprintf(errorActiveGateConflictingVolumeName, volume.Name)
		}
	}

	for _, volumeMount := range dk.Spec.ActiveGate.VolumeMounts {
		for _, managedPath := range agconsts.VolumeMountPaths {
			rel, err := filepath.Rel(managedPath, volumeMount.MountPath)
			if err != nil {
				continue
			}

			if !strings.HasPrefix(rel, "../") {
				log.Info("conflicting ActiveGate volume mount path detected", "path", volumeMount.MountPath)

				return fmt.Sprintf(errorActiveGateConflictingVolumeMountPath, volumeMount.MountPath)
			}
		}
	}

	return ""
}

// allowedInNonRootV2SCC are the volume types permitted by the OpenShift nonroot-v2 SCC.
var allowedInNonRootV2SCC = []func(corev1.VolumeSource) bool{
	func(s corev1.VolumeSource) bool { return s.ConfigMap != nil },
	func(s corev1.VolumeSource) bool { return s.CSI != nil },
	func(s corev1.VolumeSource) bool { return s.DownwardAPI != nil },
	func(s corev1.VolumeSource) bool { return s.EmptyDir != nil },
	func(s corev1.VolumeSource) bool { return s.Ephemeral != nil },
	func(s corev1.VolumeSource) bool { return s.PersistentVolumeClaim != nil },
	func(s corev1.VolumeSource) bool { return s.Projected != nil },
	func(s corev1.VolumeSource) bool { return s.Secret != nil },
	func(s corev1.VolumeSource) bool { return s.Image != nil },
}

func activeGateHasDisallowedVolumeType(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	for _, volume := range dk.Spec.ActiveGate.Volumes {
		if !slices.ContainsFunc(allowedInNonRootV2SCC, func(isAllowed func(corev1.VolumeSource) bool) bool {
			return isAllowed(volume.VolumeSource)
		}) {
			log.Info("ActiveGate volume uses a type disallowed by the OpenShift nonroot-v2 SCC", "volume", volume.Name)

			return fmt.Sprintf(errorActiveGateDisallowedVolumeType, volume.Name)
		}
	}

	return ""
}

func activeGateHasInvalidVolumes(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	if err := validateVolumes(dk.Spec.ActiveGate.Volumes, dk.Spec.ActiveGate.VolumeMounts); err != nil {
		log.Info("ActiveGate has invalid volume configuration", "error", err)

		return fmt.Sprintf(errorActiveGateInvalidVolumes, err)
	}

	return ""
}

// validateVolumes checks that:
//   - no two volumes share the same name
//   - every VolumeMount references a volume that is defined
//   - no two mounts share the same mount path
func validateVolumes(volumes []corev1.Volume, mounts []corev1.VolumeMount) error {
	definedVolumes := sets.New[string]()
	definedPaths := sets.New[string]()

	for _, v := range volumes {
		if definedVolumes.Has(v.Name) {
			return errors.New("duplicate volume name: " + v.Name)
		}

		definedVolumes.Insert(v.Name)
	}

	for _, m := range mounts {
		if !definedVolumes.Has(m.Name) {
			return errors.New("has volume mount without matching volume defined: " + m.Name)
		}

		if definedPaths.Has(m.MountPath) {
			return errors.New("duplicate volume mount path: " + m.MountPath)
		}

		definedPaths.Insert(m.MountPath)
	}

	return nil
}
