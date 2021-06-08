package csigc

import (
	"os"
	"path/filepath"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

//const (
//	maxLogFolderSize    = 300000
//	maxAmountOfLogFiles = 1000
//	maxLogAge           = 14 * 24 * time.Hour
//)

func (gc *CSIGarbageCollector) runLogGarbageCollection(versionReferences []os.FileInfo, tenantUUID string) error {
	fs := &afero.Afero{Fs: gc.fs}
	gcPodUIDs, err := gc.getPodUIDs(fs, versionReferences, tenantUUID)
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

	deadPods := difference(gcPodUIDs, logPodUIDs)

	// shouldDelete := len(deadPods) > 0 && (sizeTooBig(fs, logPath) || len(deadPods) > maxAmountOfLogFiles)
	shouldDelete := true
	if shouldDelete {
		err = gc.removeDeadLogFolders(deadPods, fs, logPath)
		if err != nil {
			gc.logger.Error(err, "failed to remove dead log directories")
			return errors.WithStack(err)
		}
	}

	return nil
}

func (gc *CSIGarbageCollector) removeDeadLogFolders(deadPods []os.FileInfo, fs *afero.Afero, logPath string) error {
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

func isOlderThanTwoWeeks(t time.Time) bool {
	//return time.Since(t) > maxLogAge
	return time.Since(t) > 5*time.Minute
}

//func sizeTooBig(fs *afero.Afero, logPath string) bool {
//	var size int64
//	_ = fs.Walk(logPath, func(_ string, info os.FileInfo, err error) error {
//		if err != nil {
//			return err
//		}
//		if !info.IsDir() {
//			size += info.Size()
//		}
//		return err
//	})
//
//	return size > maxLogFolderSize
//}

func (gc *CSIGarbageCollector) getPodUIDs(fs *afero.Afero, versionReferences []os.FileInfo, tenantUUID string) ([]os.FileInfo, error) {
	var podUIDs []os.FileInfo
	versionReferencesBase := filepath.Join(gc.opts.RootDir, dtcsi.DataPath, tenantUUID, dtcsi.GarbageCollectionPath)
	gc.logger.Info("run garbage collection for binaries", "versionReferencesBase", versionReferencesBase)

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

func difference(slice1 []os.FileInfo, slice2 []os.FileInfo) []os.FileInfo {
	var diff []os.FileInfo

	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
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
			slice1, slice2 = slice2, slice1
		}
	}

	return diff
}
