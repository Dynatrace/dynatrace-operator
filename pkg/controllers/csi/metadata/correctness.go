package metadata

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"k8s.io/mount-utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CorrectnessChecker struct {
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

	os.MkdirAll(checker.path.AppMountsBaseDir(), os.ModePerm)

	for _, appMount := range oldAppMounts {
		oldPath := filepath.Dir(appMount.Path)
		volumeID := filepath.Base(oldPath)
		newPath := checker.path.AppMountForID(volumeID)

		stat, err := os.Stat(newPath)
		if err == nil && stat.IsDir() {
			continue
		}

		err = symlink.Create(oldPath, newPath)
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

		os.MkdirAll(checker.path.DynaKubeDir(dk.Name), os.ModePerm)

		newPath := checker.path.OsAgentDir(dk.Name)

		stat, err := os.Stat(newPath)
		if err == nil && stat.IsDir() {
			continue
		}

		tenantUUID, err := TenantUUIDFromAPIURL(dk.APIURL())
		if err != nil {
			log.Error(err, "malformed APIURL for dynakube, skipping host dir migration for it", "dk", dk.Name, "apiUrl", dk.APIURL())

			continue
		}

		oldPath := checker.path.OldOsAgentDir(tenantUUID)

		_, err = os.Stat(oldPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			log.Error(err, "failed to check deprecated host dir existence, skipping host dir migration for it", "dk", dk.Name, "apiUrl", dk.APIURL())

			continue
		}

		err = symlink.Create(oldPath, newPath)
		if err != nil {
			log.Error(err, "failed to symlink old host mount to new location", "old-path", oldPath, "new-path", newPath)
		} else {
			log.Info("migrated old host mount to new location", "old-path", oldPath, "new-path", newPath)
		}
	}
}

func GetRelevantDynaKubes(ctx context.Context, apiReader client.Reader) ([]dynakube.DynaKube, error) {
	var dkList dynakube.DynaKubeList

	err := apiReader.List(ctx, &dkList, client.InNamespace(k8senv.DefaultNamespace()))
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
				switch dirType, dirPath, _ := strings.Cut(opt, "="); dirType {
				case "lowerdir":
					overlayMount.LowerDir = dirPath
				case "upperdir":
					overlayMount.UpperDir = dirPath
				case "workdir":
					overlayMount.WorkDir = dirPath
				}
			}

			relevantMounts = append(relevantMounts, overlayMount)
		}
	}

	return relevantMounts, nil
}
