package metadata

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/spf13/afero"
	"k8s.io/mount-utils"
)

type CorrectnessChecker struct {
	fs      afero.Fs
	mounter mount.Interface
	path    PathResolver
}

type OverlayMount struct {
	Path     string
	LowerDir string
	UpperDir string
	WorkDir  string
}

func NewCorrectnessChecker(opts dtcsi.CSIOptions) *CorrectnessChecker {
	return &CorrectnessChecker{
		fs:      afero.NewOsFs(),
		mounter: mount.New(""),
		path:    PathResolver{RootDir: opts.RootDir},
	}
}

// CorrectMetadata checks if the entries in the storage are actually valid
// Removes not valid entries
// "Moves" agent bins from deprecated location. (just creates a symlink)
func (checker *CorrectnessChecker) CorrectCSI(ctx context.Context) error {
	checker.migrateAppMounts()

	return nil
}

func (checker *CorrectnessChecker) migrateAppMounts() {
	baseDir := checker.path.RootDir

	appMounts, err := GetRelevantOverlayMounts(checker.mounter, baseDir)
	if err != nil {
		log.Error(err, "failed to get relevant overlay mounts")
	}

	oldAppMounts := []OverlayMount{}

	for _, appMount := range appMounts {
		if !strings.HasPrefix(appMount.Path, checker.path.AppMountsBaseDir()) {
			oldAppMounts = append(oldAppMounts, appMount)
		}
	}

	checker.fs.MkdirAll(checker.path.AppMountsBaseDir(), os.ModePerm)

	for _, appMount := range oldAppMounts {
		oldPath := filepath.Dir(appMount.Path)
		volumeID := filepath.Base(oldPath)
		newPath := checker.path.AppMountForID(volumeID)

		if folderExists(checker.fs, newPath) {
			continue
		}

		linker, ok := checker.fs.(afero.Linker)
		if ok { // will only be !ok during unit testing
			err := linker.SymlinkIfPossible(oldPath, newPath)
			if err != nil {
				log.Error(err, "failed to symlink old app mount to new location", "old-path", oldPath, "new-path", newPath)
			} else {
				log.Info("migrated old app mount to new location", "old-path", oldPath, "new-path", newPath)
			}
		}
	}
}

func folderExists(fs afero.Fs, filename string) bool {
	info, err := fs.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return info.IsDir()
}

func GetRelevantOverlayMounts(mounter mount.Interface, baseFolder string) ([]OverlayMount, error) {
	mountPoints, err := mounter.List()
	if err != nil {
		return nil, err
	}

	relevantMounts := []OverlayMount{}

	for _, mountPoint := range mountPoints {
		if mountPoint.Device == "overlay" {
			if !strings.HasPrefix(mountPoint.Path, baseFolder) {
				continue
			}

			overlayMount := OverlayMount{
				Path: mountPoint.Path,
			}

			for _, opt := range mountPoint.Opts {
				switch {
				case strings.HasPrefix(opt, "lowerdir="):
					split := strings.Split(opt, "=")
					overlayMount.LowerDir = split[1]
				case strings.HasPrefix(opt, "upperdir="):
					split := strings.Split(opt, "=")
					overlayMount.UpperDir = split[1]
				case strings.HasPrefix(opt, "workdir="):
					split := strings.Split(opt, "=")
					overlayMount.WorkDir = split[1]
				}
			}

			relevantMounts = append(relevantMounts, overlayMount)
		}
	}

	return relevantMounts, nil
}
