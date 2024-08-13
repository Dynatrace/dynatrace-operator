package csigc

import (
	"os"
	"strconv"
	"time"

	"github.com/spf13/afero"
)

const (
	defaultMaxUnmountedCsiVolumeAge = 7 * 24 * time.Hour
	maxUnmountedCsiVolumeAgeEnv     = "MAX_UNMOUNTED_VOLUME_AGE"
)

func (gc *CSIGarbageCollector) runUnmountedVolumeGarbageCollection(tenantUUID string) {
	unmountedVolumes, err := gc.getUnmountedVolumes(tenantUUID)
	if err != nil {
		log.Info("failed to get unmounted volume information", "error", err)

		return
	}

	gc.removeUnmountedVolumesIfNecessary(unmountedVolumes, tenantUUID)
}

func (gc *CSIGarbageCollector) getUnmountedVolumes(tenantUUID string) ([]os.FileInfo, error) {
	var unusedVolumeIDs []os.FileInfo

	mountsDirectoryPath := gc.path.AgentRunDir(tenantUUID)

	volumeIDs, err := afero.ReadDir(gc.fs, mountsDirectoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("no mount directories found for this tenant, moving on", "tenantUUID", tenantUUID, "path", mountsDirectoryPath)

			return nil, nil
		}

		return nil, err
	}

	for _, volumeID := range volumeIDs {
		mappedDir := gc.path.OverlayMappedDir(tenantUUID, volumeID.Name())
		isUnused, err := afero.IsEmpty(gc.fs, mappedDir)

		if err != nil {
			log.Info("failed to check if directory is empty, skipping", "folder", mappedDir, "error", err)

			continue
		}

		if isUnused {
			unusedVolumeIDs = append(unusedVolumeIDs, volumeID)
		}
	}

	return unusedVolumeIDs, nil
}

func (gc *CSIGarbageCollector) removeUnmountedVolumesIfNecessary(unusedVolumeIDs []os.FileInfo, tenantUUID string) {
	for _, unusedVolumeID := range unusedVolumeIDs {
		if gc.isUnmountedVolumeTooOld(unusedVolumeID.ModTime()) {
			err := gc.fs.RemoveAll(gc.path.AgentRunDirForVolume(tenantUUID, unusedVolumeID.Name()))
			if err != nil {
				log.Info("failed to remove logs for pod", "podUID", unusedVolumeID.Name(), "error", err)
			}
		}
	}
}

func (gc *CSIGarbageCollector) isUnmountedVolumeTooOld(t time.Time) bool {
	return gc.maxUnmountedVolumeAge == 0 || time.Since(t) > gc.maxUnmountedVolumeAge
}

func determineMaxUnmountedVolumeAge(maxAgeEnvValue string) time.Duration {
	if maxAgeEnvValue == "" {
		return defaultMaxUnmountedCsiVolumeAge
	}

	maxAge, err := strconv.Atoi(maxAgeEnvValue)
	if err != nil {
		log.Error(err, "failed to parse MaxUnmountedCsiVolumeAge from", "env", maxUnmountedCsiVolumeAgeEnv, "value", maxAgeEnvValue)

		return defaultMaxUnmountedCsiVolumeAge
	}

	if maxAge <= 0 {
		log.Info("max unmounted csi volume age is set to 0, files will be deleted immediately")

		return 0
	}

	log.Info("max unmounted csi volume age used", "age in days", maxAge)

	return 24 * time.Duration(maxAge) * time.Hour
}
