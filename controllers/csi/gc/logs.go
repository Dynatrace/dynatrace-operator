package csigc

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

const (
	maxLogFolderSizeBytes = 300000
	maxNumberOfLogFiles   = 1000
	maxLogAge             = 14 * 24 * time.Hour
)

type logFileInfo struct {
	UnusedVolumeIDs []os.FileInfo
	NumberOfFiles   int64
	OverallSize     int64
}

func (gc *CSIGarbageCollector) runLogGarbageCollection(tenantUUID string) {
	agentDirectoryForPod := filepath.Join(gc.opts.RootDir, tenantUUID, "run")
	logs, err := gc.getLogFileInfo(agentDirectoryForPod)
	if err != nil {
		gc.logger.Info("failed to get log file information")
		return
	}

	gc.removeLogsIfNecessary(logs, maxLogFolderSizeBytes, maxNumberOfLogFiles, agentDirectoryForPod)
}

func (gc *CSIGarbageCollector) removeLogsIfNecessary(logs *logFileInfo, maxSize int64, maxFile int64, agentDirectoryForPod string) {
	shouldDelete := logs.NumberOfFiles > 0 && (logs.OverallSize > maxSize || logs.NumberOfFiles > maxFile)
	if shouldDelete {
		gc.tryRemoveLogFolders(logs.UnusedVolumeIDs, agentDirectoryForPod)
	}
}

func (gc *CSIGarbageCollector) getLogFileInfo(agentDirectoryForPod string) (*logFileInfo, error) {
	unusedVolumeIDs, err := gc.getUnusedVolumeIDs(agentDirectoryForPod)
	if err != nil {
		return nil, err
	}

	var nrOfFiles int64
	var size int64
	for _, volumeID := range unusedVolumeIDs {
		_ = afero.Walk(gc.fs, filepath.Join(agentDirectoryForPod, volumeID.Name(), "var"), func(_ string, file os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !file.IsDir() {
				nrOfFiles++
				size += file.Size()
			}
			return nil
		})
	}

	return &logFileInfo{
		UnusedVolumeIDs: unusedVolumeIDs,
		NumberOfFiles:   nrOfFiles,
		OverallSize:     size,
	}, nil
}

func (gc *CSIGarbageCollector) getUnusedVolumeIDs(agentDirectoryForPod string) ([]os.FileInfo, error) {
	var unusedVolumeIDs []os.FileInfo

	volumeIDs, err := afero.ReadDir(gc.fs, agentDirectoryForPod)
	if err != nil {
		return nil, err
	}

	for _, volumeID := range volumeIDs {
		unused, err := afero.IsEmpty(gc.fs, filepath.Join(agentDirectoryForPod, volumeID.Name(), "mapped"))
		if err != nil {
			return nil, err
		}

		if unused {
			unusedVolumeIDs = append(unusedVolumeIDs, volumeID)
		}
	}

	return unusedVolumeIDs, nil
}

func (gc *CSIGarbageCollector) tryRemoveLogFolders(unusedVolumeIDs []os.FileInfo, agentDirectoryForPod string) {
	for _, unusedVolumeID := range unusedVolumeIDs {
		if isOlderThanTwoWeeks(unusedVolumeID.ModTime()) {
			if err := gc.fs.RemoveAll(filepath.Join(agentDirectoryForPod, unusedVolumeID.Name())); err != nil {
				gc.logger.Info("failed to remove logs for pod", "podUID", unusedVolumeID.Name(), "error", err)
			}
		}
	}
}

func isOlderThanTwoWeeks(t time.Time) bool {
	return time.Since(t) > maxLogAge
}
