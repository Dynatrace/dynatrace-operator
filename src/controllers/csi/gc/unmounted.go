package csigc

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/spf13/afero"
)

const (
	defaultMaxUnmountedCsiVolumeAge = 14 * 24 * time.Hour
	MaxUnmountedCsiVolumeAgeEnv     = "MAX_UNMOUNTED_VOLUME_AGE"
)

func (gc *CSIGarbageCollector) runUnmountedVolumeGarbageCollection(ctx context.Context, tenantUUID string) {
	unmountedVolumes, err := gc.getUnmountedVolumes(tenantUUID)
	if err != nil {
		log.Info("failed to get unmounted volume information")
		return
	}
	gc.removeUnmountedVolumesIfNecessary(unmountedVolumes, tenantUUID)
}

func (gc *CSIGarbageCollector) getUnmountedVolumes(tenantUUID string) ([]os.FileInfo, error) {
	var unusedVolumeIDs []os.FileInfo

	volumeIDs, err := afero.ReadDir(gc.fs, gc.path.AgentRunDir(tenantUUID))
	if err != nil {
		return nil, err
	}

	for _, volumeID := range volumeIDs {
		unused, err := afero.IsEmpty(gc.fs, gc.path.OverlayMappedDir(tenantUUID, volumeID.Name()))
		if err != nil {
			return nil, err
		}

		if unused {
			unusedVolumeIDs = append(unusedVolumeIDs, volumeID)
		}
	}

	return unusedVolumeIDs, nil
}

func (gc *CSIGarbageCollector) removeUnmountedVolumesIfNecessary(unusedVolumeIDs []os.FileInfo, tenantUUID string) {
	for _, unusedVolumeID := range unusedVolumeIDs {
		if gc.isUnmountedVolumeTooOld(unusedVolumeID.ModTime()) {
			if err := gc.fs.RemoveAll(gc.path.AgentRunDirForVolume(tenantUUID, unusedVolumeID.Name())); err != nil {
				log.Info("failed to remove logs for pod", "podUID", unusedVolumeID.Name(), "error", err)
			}
		}
	}
}

func (gc *CSIGarbageCollector) isUnmountedVolumeTooOld(t time.Time) bool {
	return gc.maxUnmountedVolumeAge == 0 || time.Since(t) > gc.maxUnmountedVolumeAge
}

func determineMaxUnmountedVolumeAge() time.Duration {
	maxAgeEnv := os.Getenv(MaxUnmountedCsiVolumeAgeEnv)
	if maxAgeEnv == "" {
		return defaultMaxUnmountedCsiVolumeAge
	}

	maxAge, err := strconv.Atoi(maxAgeEnv)
	if err != nil {
		log.Error(err, "failed to parse MaxUnmountedCsiVolumeAge from", "env", MaxUnmountedCsiVolumeAgeEnv, "value", maxAgeEnv)
		return defaultMaxUnmountedCsiVolumeAge
	}
	if maxAge < 0 {
		return 0
	}
	return 24 * time.Duration(maxAge) * time.Hour
}
