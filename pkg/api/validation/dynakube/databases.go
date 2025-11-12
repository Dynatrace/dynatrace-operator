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
	errorFailedToValidateVolumeMount      = `Failed to validate database volume mount at %s.`

	warningHostPathDatabaseVolumeDetected = `Host path database volume detected. If you're on OCP, mounting host path volumes will be prohibited by the SCC and cause silent failures. If still want to do this, make sure to create and bind corresponding roles.`
)

func conflictingOrInvalidDatabasesVolumeMounts(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.Extensions().IsDatabasesEnabled() {
		return ""
	}

	defaultVolumeMounts := databases.GetDefaultVolumeMounts(dk)
	for _, database := range dk.Spec.Extensions.Databases {
		for _, volumeMount := range database.VolumeMounts {
			volumeFound := slices.IndexFunc(database.Volumes, func(volume corev1.Volume) bool {
				return volume.Name == volumeMount.Name
			}) >= 0
			if !volumeFound {
				log.Info("invalid database volume mount detected; no matching volume found", "volume", volumeMount.Name)

				return fmt.Sprintf(errorInvalidDatabasesVolumeMount, volumeMount.Name)
			}

			for _, defaultVolumeMount := range defaultVolumeMounts {
				rel, err := filepath.Rel(defaultVolumeMount.MountPath, volumeMount.MountPath)
				if err != nil {
					return fmt.Sprintf(errorFailedToValidateVolumeMount, volumeMount.MountPath)
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
				log.Info("host path volume detected, this may cause issues on OCP", "volumeName", volume.Name)

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
			volumeUsed := false

			for _, volumeMount := range database.VolumeMounts {
				if volumeMount.Name == volume.Name {
					volumeUsed = true

					break
				}
			}

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
