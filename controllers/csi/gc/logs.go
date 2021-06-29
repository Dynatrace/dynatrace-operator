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
	fs := &afero.Afero{Fs: gc.fs}
	logs, err := gc.getLogFileInfo(tenantUUID, fs)
	if err != nil {
		gc.logger.Info("failed to get log file information")
		return
	}

	shouldDelete := logs.NumberOfFiles > 0 && (logs.OverallSize > maxLogFolderSizeBytes || logs.NumberOfFiles > maxNumberOfLogFiles)
	if shouldDelete {
		gc.tryRemoveLogFolders(logs.UnusedVolumeIDs, tenantUUID, fs)
	}
}

func (gc *CSIGarbageCollector) getLogFileInfo(tenantUUID string, fs *afero.Afero) (*logFileInfo, error) {
	unusedVolumeIDs, err := gc.getUnusedVolumeIDs(tenantUUID, fs)
	if err != nil {
		return nil, err
	}

	var nrOfFiles int64
	var size int64
	for _, volumeID := range unusedVolumeIDs {
		_ = fs.Walk(filepath.Join(gc.opts.RootDir, tenantUUID, "run", volumeID.Name(), "var"), func(_ string, file os.FileInfo, err error) error {
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

func (gc *CSIGarbageCollector) getUnusedVolumeIDs(tenantUUID string, fs *afero.Afero) ([]os.FileInfo, error) {
	var unusedVolumeIDs []os.FileInfo

	agentDirectoryForPod := filepath.Join(gc.opts.RootDir, tenantUUID, "run")
	volumeIDs, err := fs.ReadDir(agentDirectoryForPod)
	if err != nil {
		return nil, err
	}

	for _, volumeID := range volumeIDs {
		unused, err := fs.IsEmpty(filepath.Join(agentDirectoryForPod, volumeID.Name(), "mapped"))
		if err != nil {
			return nil, err
		}

		if unused {
			unusedVolumeIDs = append(unusedVolumeIDs, volumeID)
		}
	}

	return unusedVolumeIDs, nil
}

func (gc *CSIGarbageCollector) tryRemoveLogFolders(unusedVolumeIDs []os.FileInfo, tenantUUID string, fs *afero.Afero) {
	for _, unusedVolumeID := range unusedVolumeIDs {
		if isOlderThanTwoWeeks(unusedVolumeID.ModTime()) {
			if err := fs.RemoveAll(filepath.Join(gc.opts.RootDir, tenantUUID, "run", unusedVolumeID.Name())); err != nil {
				gc.logger.Info("failed to remove logs for pod", "podUID", unusedVolumeID.Name(), "error", err)
			}
		}
	}
}

func isOlderThanTwoWeeks(t time.Time) bool {
	return time.Since(t) > maxLogAge
}
