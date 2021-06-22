package csigc

import (
	"os"
	"path/filepath"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const (
	maxLogFolderSizeBytes = 300000
	maxNumberOfLogFiles   = 1000
	maxLogAge             = 14 * 24 * time.Hour
)

func (gc *CSIGarbageCollector) runLogGarbageCollection(versionReferences []os.FileInfo, tenantUUID string) error {
	fs := &afero.Afero{Fs: gc.fs}
	gcPodUIDs, err := gc.getCurrentlyUsedPodReferencePodUIDs(fs, versionReferences, tenantUUID)
	if err != nil {
		gc.logger.Info("skipped, failed to get podUIDs")
		return errors.WithStack(err)
	}

	logPath := filepath.Join(gc.opts.RootDir, dtcsi.DataPath, tenantUUID, dtcsi.LogDir)
	logPodUIDs, err := fs.ReadDir(logPath)
	if err != nil {
		gc.logger.Info("skipped, failed to get references")
		return errors.WithStack(err)
	}

	deadPods := getDeadPodUIDsDelta(gcPodUIDs, logPodUIDs)

	nrOfLogFiles := gc.getNumberOfLogFiles(deadPods, fs, logPath)

	shouldDelete := len(deadPods) > 0 && (sizeTooBig(fs, logPath) || nrOfLogFiles > maxNumberOfLogFiles)
	if shouldDelete {
		err = gc.tryRemoveLogFolders(deadPods, fs, logPath)
		if err != nil {
			gc.logger.Error(err, "failed to remove dead log directories")
			return errors.WithStack(err)
		}
	}

	return nil
}

func (gc *CSIGarbageCollector) getNumberOfLogFiles(deadPods []os.FileInfo, fs *afero.Afero, logPath string) int64 {
	var nrOfFiles int64
	for _, podUID := range deadPods {
		_ = fs.Walk(filepath.Join(logPath, podUID.Name()), func(_ string, file os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !file.IsDir() {
				nrOfFiles++
			}
			return err
		})
	}

	return nrOfFiles
}

func (gc *CSIGarbageCollector) tryRemoveLogFolders(deadPods []os.FileInfo, fs *afero.Afero, logPath string) error {
	for _, deadPod := range deadPods {
		if isOlderThanTwoWeeks(deadPod.ModTime()) {
			if err := fs.RemoveAll(filepath.Join(logPath, deadPod.Name())); err != nil {
				gc.logger.Error(err, "failed to remove logs for pod", "podUID", deadPod.Name())
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

func (gc *CSIGarbageCollector) getCurrentlyUsedPodReferencePodUIDs(fs *afero.Afero, versionReferences []os.FileInfo, tenantUUID string) ([]os.FileInfo, error) {
	var podUIDs []os.FileInfo
	versionReferencesBase := filepath.Join(gc.opts.RootDir, dtcsi.DataPath, tenantUUID, dtcsi.GarbageCollectionPath)

	for _, fileInfo := range versionReferences {
		references := filepath.Join(versionReferencesBase, fileInfo.Name())

		podReferences, err := fs.ReadDir(references)
		if err != nil {
			gc.logger.Info("skipped, failed to get references")
			return nil, errors.WithStack(err)
		}

		podUIDs = append(podUIDs, podReferences...)
	}

	return podUIDs, nil
}

func isOlderThanTwoWeeks(t time.Time) bool {
	return time.Since(t) > maxLogAge
}

func sizeTooBig(fs *afero.Afero, logPath string) bool {
	var size int64
	_ = fs.Walk(logPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})

	return size > maxLogFolderSizeBytes
}

func getDeadPodUIDsDelta(currentlyUsedPodReferencePodUIDs []os.FileInfo, logPodUIDs []os.FileInfo) []os.FileInfo {
	var diff []os.FileInfo

	for i := 0; i < 2; i++ {
		for _, s1 := range currentlyUsedPodReferencePodUIDs {
			found := false
			for _, s2 := range logPodUIDs {
				if s1 == s2 {
					found = true
					break
				}
			}

			if !found {
				diff = append(diff, s1)
			}
		}

		if i == 0 {
			currentlyUsedPodReferencePodUIDs, logPodUIDs = logPodUIDs, currentlyUsedPodReferencePodUIDs
		}
	}

	return diff
}
