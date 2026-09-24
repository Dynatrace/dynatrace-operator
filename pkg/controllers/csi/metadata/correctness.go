// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
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
	Path      string
	UpperDir  string
	WorkDir   string
	LowerDirs []string
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
	ctx, _ = logd.NewFromContext(ctx, "csi-metadata")
	checker.migrateAppMounts(ctx)
	checker.migrateHostMounts(ctx)

	return nil
}

func (checker *CorrectnessChecker) migrateAppMounts(ctx context.Context) {
	log := logd.FromContext(ctx)
	baseDir := checker.path.RootDir

	appMounts, err := GetOverlayMountsIn(ctx, checker.mounter, baseDir)
	if err != nil {
		log.Error(err, "failed to get relevant overlay mounts")
	}

	oldAppMounts := []OverlayMount{}

	for _, appMount := range appMounts {
		if !strings.HasPrefix(appMount.Path, checker.path.AppMountsBaseDir()) {
			oldAppMounts = append(oldAppMounts, appMount)
		}
	}

	err = os.MkdirAll(checker.path.AppMountsBaseDir(), dtcsi.AppmountsDirPermissions)
	if err != nil {
		log.Error(err, "failed to create app mounts base directory")
	} else {
		err = os.Chmod(checker.path.AppMountsBaseDir(), dtcsi.AppmountsDirPermissions)
		if err != nil {
			log.Error(err, "failed to set permissions of the app mounts base directory")
		}
	}

	for _, appMount := range oldAppMounts {
		oldPath := filepath.Dir(appMount.Path)
		volumeID := filepath.Base(oldPath)
		newPath := checker.path.AppMountForID(volumeID)

		stat, err := os.Stat(newPath)
		if err == nil && stat.IsDir() {
			continue
		}

		err = symlink.Create(ctx, oldPath, newPath)
		if err != nil {
			log.Error(err, "failed to symlink old app mount to new location", "old-path", oldPath, "new-path", newPath)
		} else {
			log.Info("migrated old app mount to new location", "old-path", oldPath, "new-path", newPath)
		}
	}
}

func (checker *CorrectnessChecker) migrateHostMounts(ctx context.Context) {
	log := logd.FromContext(ctx)

	dks, err := GetRelevantDynaKubes(ctx, checker.apiReader)
	if err != nil {
		log.Error(err, "failed to list the available dynakubes, skipping host mount migration")

		return
	}

	for _, dk := range dks {
		if !dk.OneAgent().IsReadOnlyFSSupported() {
			continue
		}

		err := os.MkdirAll(checker.path.DynaKubeDir(dk.Name), os.ModePerm)
		if err != nil {
			log.Error(err, "failed to create dynakube directory")
		}

		newPath := checker.path.OSAgentDir(dk.Name)

		stat, err := os.Stat(newPath)
		if err == nil && stat.IsDir() {
			continue
		}

		tenantUUID, err := TenantUUIDFromAPIURL(dk.APIURL())
		if err != nil {
			log.Error(err, "malformed APIURL for dynakube, skipping host dir migration for it", "dynakube", dk.Name, "apiUrl", dk.APIURL())

			continue
		}

		oldPath := checker.path.OldOSAgentDir(tenantUUID)

		_, err = os.Stat(oldPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			log.Error(err, "failed to check deprecated host dir existence, skipping host dir migration for it", "dynakube", dk.Name, "apiUrl", dk.APIURL())

			continue
		}

		err = symlink.Create(ctx, oldPath, newPath)
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

// GetOverlayMountsIn returns the overlay mounts that are mounted somewhere under baseFolder.
func GetOverlayMountsIn(ctx context.Context, mounter mount.Interface, baseFolder string) ([]OverlayMount, error) {
	return collectOverlayMounts(ctx, mounter, func(overlayMount OverlayMount) bool {
		ok, _ := isSubfolder(overlayMount.Path, baseFolder)
		return ok
	})
}

// GetOverlayMountsWithLowerDirIn returns the overlay mounts that use at least one lower directory
// under baseFolder, regardless of where the mount point itself is.
//
// App mounts are mounted directly at the kubelet target path (/var/lib/kubelet/pods/...), so the
// lower directory is the only part of such a mount that points back into the CSI filesystem.
func GetOverlayMountsWithLowerDirIn(ctx context.Context, mounter mount.Interface, baseFolder string) ([]OverlayMount, error) {
	return collectOverlayMounts(ctx, mounter, func(overlayMount OverlayMount) bool {
		return slices.ContainsFunc(overlayMount.LowerDirs, func(lowerDir string) bool {
			ok, _ := isSubfolder(lowerDir, baseFolder)
			return ok
		})
	})
}

func collectOverlayMounts(ctx context.Context, mounter mount.Interface, isRelevant func(OverlayMount) bool) ([]OverlayMount, error) {
	log := logd.FromContext(ctx)

	// Only lists the mounts of our own mount namespace, so this needs HostToContainer propagation
	// on the kubelet path (/var/lib/kubelet) to see the app mounts the CSI server creates.
	mountPoints, err := mounter.List()
	if err != nil {
		return nil, err
	}

	relevantMounts := []OverlayMount{}

	for _, mountPoint := range mountPoints {
		if mountPoint.Device != "overlay" {
			continue
		}

		overlayMount := parseOverlayMount(mountPoint)
		isRelevantMount := isRelevant(overlayMount)

		log.Debug("checked overlay mount",
			"path", overlayMount.Path,
			"lowerDirs", overlayMount.LowerDirs,
			"upperDir", overlayMount.UpperDir,
			"relevant", isRelevantMount)

		if isRelevantMount {
			relevantMounts = append(relevantMounts, overlayMount)
		}
	}

	return relevantMounts, nil
}

func parseOverlayMount(mountPoint mount.MountPoint) OverlayMount {
	overlayMount := OverlayMount{
		Path: mountPoint.Path,
	}

	for _, opt := range mountPoint.Opts {
		switch dirType, dirPath, _ := strings.Cut(opt, "="); dirType {
		// "lowerdir+" is how kernels >= 6.7 spell the additional lower layers.
		case "lowerdir", "lowerdir+":
			overlayMount.LowerDirs = append(overlayMount.LowerDirs, splitLowerDirs(dirPath)...)
		case "upperdir":
			overlayMount.UpperDir = dirPath
		case "workdir":
			overlayMount.WorkDir = dirPath
		}
	}

	return overlayMount
}

// splitLowerDirs splits the value of an overlayfs lowerdir option into the individual layers.
// Layers are separated by ':', and overlayfs escapes ':', ',' and the backslash itself within a
// path with a leading backslash.
func splitLowerDirs(value string) []string {
	var (
		lowerDirs []string
		current   strings.Builder
		escaped   bool
	)

	appendCurrent := func() {
		if current.Len() > 0 {
			lowerDirs = append(lowerDirs, current.String())
			current.Reset()
		}
	}

	for _, char := range value {
		switch {
		case escaped:
			current.WriteRune(char)

			escaped = false
		case char == '\\':
			escaped = true
		case char == ':':
			appendCurrent()
		default:
			current.WriteRune(char)
		}
	}

	appendCurrent()

	return lowerDirs
}

func isSubfolder(child, parent string) (bool, error) {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false, err
	}
	return !strings.HasPrefix(rel, ".."), nil
}
