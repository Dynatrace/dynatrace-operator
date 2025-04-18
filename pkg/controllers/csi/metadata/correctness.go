package metadata

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/spf13/afero"
	"k8s.io/mount-utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CorrectnessChecker struct {
	fs        afero.Afero
	apiReader client.Reader
	mounter   mount.Interface
	path      PathResolver
}

type OverlayMount struct {
	Path     string
	LowerDir string
	UpperDir string
	WorkDir  string
}

func NewCorrectnessChecker(apiReader client.Reader, opts dtcsi.CSIOptions) *CorrectnessChecker {
	return &CorrectnessChecker{
		apiReader: apiReader,
		fs:        afero.Afero{Fs: afero.NewOsFs()},
		mounter:   mount.New(""),
		path:      PathResolver{RootDir: opts.RootDir},
	}
}

// CorrectMetadata checks if the entries in the storage are actually valid
// Removes not valid entries
// "Moves" agent bins from deprecated location. (just creates a symlink)
func (checker *CorrectnessChecker) CorrectCSI(ctx context.Context) error {
	checker.migrateAppMounts()
	checker.migrateHostMounts(ctx)

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

		exists, _ := checker.fs.DirExists(newPath)
		if exists {
			continue
		}

		err := symlink.Create(checker.fs.Fs, oldPath, newPath)
		if err != nil {
			log.Error(err, "failed to symlink old app mount to new location", "old-path", oldPath, "new-path", newPath)
		} else {
			log.Info("migrated old app mount to new location", "old-path", oldPath, "new-path", newPath)
		}
	}
}

func (checker *CorrectnessChecker) migrateHostMounts(ctx context.Context) {
	dks, err := GetRelevantDynaKubes(ctx, checker.apiReader)
	if err != nil {
		log.Error(err, "failed to list the available dynakubes, skipping host mount migration")

		return
	}

	for _, dk := range dks {
		if !dk.OneAgent().IsReadOnlyFSSupported() {
			continue
		}

		checker.fs.MkdirAll(checker.path.DynaKubeDir(dk.Name), os.ModePerm)

		newPath := checker.path.OsAgentDir(dk.Name)

		newExists, _ := checker.fs.DirExists(newPath)
		if newExists {
			continue
		}

		tenantUUID, err := TenantUUIDFromApiUrl(dk.ApiUrl())
		if err != nil {
			log.Error(err, "malformed ApiUrl for dynakube, skipping host dir migration for it", "dk", dk.Name, "apiUrl", dk.ApiUrl())

			continue
		}

		oldPath := checker.path.OldOsAgentDir(tenantUUID)

		oldExists, err := checker.fs.DirExists(oldPath)
		if err != nil {
			log.Error(err, "failed to check deprecated host dir existence, skipping host dir migration for it", "dk", dk.Name, "apiUrl", dk.ApiUrl())

			continue
		}

		if !oldExists {
			continue
		}

		err = symlink.Create(checker.fs.Fs, oldPath, newPath)
		if err != nil {
			log.Error(err, "failed to symlink old host mount to new location", "old-path", oldPath, "new-path", newPath)
		} else {
			log.Info("migrated old host mount to new location", "old-path", oldPath, "new-path", newPath)
		}
	}
}

func GetRelevantDynaKubes(ctx context.Context, apiReader client.Reader) ([]dynakube.DynaKube, error) {
	var dkList dynakube.DynaKubeList

	err := apiReader.List(ctx, &dkList, client.InNamespace(env.DefaultNamespace()))
	if err != nil {
		return nil, err
	}

	var relevantDks []dynakube.DynaKube

	for _, dk := range dkList.Items {
		if dk.OneAgent().IsAppInjectionNeeded() || dk.OneAgent().IsReadOnlyFSSupported() {
			relevantDks = append(relevantDks, dk)
		}
	}

	return relevantDks, nil
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
					split := strings.SplitN(opt, "=", 2)
					overlayMount.LowerDir = split[1]
				case strings.HasPrefix(opt, "upperdir="):
					split := strings.SplitN(opt, "=", 2)
					overlayMount.UpperDir = split[1]
				case strings.HasPrefix(opt, "workdir="):
					split := strings.SplitN(opt, "=", 2)
					overlayMount.WorkDir = split[1]
				}
			}

			relevantMounts = append(relevantMounts, overlayMount)
		}
	}

	return relevantMounts, nil
}
