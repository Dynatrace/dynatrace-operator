package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/databases"
	corev1 "k8s.io/api/core/v1"
)

const (
	errorConflictingDatabasesVolumeMounts = `Database volume mount (%s) in conflict with a default database volume mount (%s) detected. Please make sure to avoid such conflicts.`
	errorInvalidDatabasesVolumeMount      = `Invalid database volume mount detected: %s. No matching database volume found.`
	errorUnusedDatabasesVolumes           = `Unused database volume(s) found (%s). Make sure to mount all database volumes defined in the DynaKube.`
	errorInvalidDatabasesVolumeMountPath  = `Invalid database volume mount path detected (%s). Make sure to use absolute paths.`

	warningHostPathDatabaseVolumeDetected = `Host path database volume detected. If you're on OpenShift, mounting host path volumes will be prohibited by the SCC and cause silent failures. If you still want to do this, make sure to create and bind corresponding roles.`
)

func conflictingOrInvalidDatabasesVolumeMounts(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.Extensions().IsDatabasesEnabled() {
		return ""
	}

	defaultVolumeMounts := databases.GetDefaultVolumeMounts(dk)
	for _, database := range dk.Spec.Extensions.Databases {
		for _, volumeMount := range database.VolumeMounts {
			volumeFound := slices.ContainsFunc(database.Volumes, func(volume corev1.Volume) bool {
				return volume.Name == volumeMount.Name
			})
			if !volumeFound {
				log.Info("invalid database volume mount detected; no matching volume found", "volume", volumeMount.Name)

				return fmt.Sprintf(errorInvalidDatabasesVolumeMount, volumeMount.Name)
			}

			for _, defaultVolumeMount := range defaultVolumeMounts {
				rel, err := filepath.Rel(defaultVolumeMount.MountPath, volumeMount.MountPath)
				if err != nil {
					return fmt.Sprintf(errorInvalidDatabasesVolumeMountPath, volumeMount.MountPath)
				}

				if !strings.HasPrefix(rel, "../") {
					log.Info("conflicting database volume mount detected; clashes with pre-defined volume", "path", volumeMount.MountPath)

					return fmt.Sprintf(errorConflictingDatabasesVolumeMounts, volumeMount.MountPath, defaultVolumeMount.MountPath)
				}
			}
		}
	}

	return ""
}

func hostPathDatabaseVolumeFound(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.Extensions().IsDatabasesEnabled() {
		return ""
	}

	for _, database := range dk.Spec.Extensions.Databases {
		for _, volume := range database.Volumes {
			if volume.HostPath != nil {
				log.Info("host path volume detected, this may cause issues on openshift", "volumeName", volume.Name)

				return warningHostPathDatabaseVolumeDetected
			}
		}
	}

	return ""
}

func unusedDatabasesVolume(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.Extensions().IsDatabasesEnabled() {
		return ""
	}

	unusedVolumes := make([]string, 0)

	for _, database := range dk.Spec.Extensions.Databases {
		for _, volume := range database.Volumes {
			volumeUsed := slices.ContainsFunc(database.VolumeMounts, func(volumeMount corev1.VolumeMount) bool {
				return volumeMount.Name == volume.Name
			})

			if !volumeUsed {
				log.Info("unmounted database volume detected", "volume", volume.Name)
				unusedVolumes = append(unusedVolumes, volume.Name)
			}
		}
	}

	if len(unusedVolumes) > 0 {
		return fmt.Sprintf(errorUnusedDatabasesVolumes, strings.Join(unusedVolumes, ", "))
	}

	return ""
}
