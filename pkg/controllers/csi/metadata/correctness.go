package metadata

import (
	"context"
	"os"
	"path/filepath"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CorrectnessChecker struct {
	apiReader client.Reader
	fs        afero.Fs
	path      PathResolver
	access    Access
}

func NewCorrectnessChecker(cl client.Reader, access Access, opts dtcsi.CSIOptions) *CorrectnessChecker {
	return &CorrectnessChecker{
		apiReader: cl,
		fs:        afero.NewOsFs(),
		path:      PathResolver{RootDir: opts.RootDir},
		access:    access,
	}
}

// CorrectMetadata checks if the entries in the storage are actually valid
// Removes not valid entries
// "Moves" agent bins from deprecated location. (just creates a symlink)
func (checker *CorrectnessChecker) CorrectCSI(ctx context.Context) error {
	defer LogAccessOverview(checker.access)

	if err := checker.removeVolumesForMissingPods(ctx); err != nil {
		return err
	}

	if err := checker.removeMissingDynakubes(ctx); err != nil {
		return err
	}

	if err := checker.copyCodeModulesFromDeprecatedBin(ctx); err != nil {
		return err
	}

	return nil
}

// Removes volume entries if their pod is no longer exists
func (checker *CorrectnessChecker) removeVolumesForMissingPods(ctx context.Context) error {
	if checker.apiReader == nil {
		log.Info("no kubernetes client configured, skipping orphaned volume metadata cleanup")

		return nil
	}

	podNames, err := checker.access.GetPodNames(ctx)
	if err != nil {
		return err
	}

	pruned := []string{}

	for podName := range podNames {
		var pod corev1.Pod
		if err := checker.apiReader.Get(ctx, client.ObjectKey{Name: podName}, &pod); !k8serrors.IsNotFound(err) {
			continue
		}

		volumeID := podNames[podName]
		if err := checker.access.DeleteVolume(ctx, volumeID); err != nil {
			return err
		}

		pruned = append(pruned, volumeID+"|"+podName)
	}

	log.Info("CSI volumes database is corrected for missing pods (volume|pod)", "prunedRows", pruned)

	return nil
}

// Removes dynakube entries if their Dynakube instance no longer exists in the cluster
func (checker *CorrectnessChecker) removeMissingDynakubes(ctx context.Context) error {
	if checker.apiReader == nil {
		log.Info("no kubernetes client configured, skipping orphaned dynakube metadata cleanup")

		return nil
	}

	dynakubes, err := checker.access.GetTenantsToDynakubes(ctx)
	if err != nil {
		return err
	}

	pruned := []string{}

	for dynakubeName := range dynakubes {
		var dynakube dynatracev1beta1.DynaKube
		if err := checker.apiReader.Get(ctx, client.ObjectKey{Name: dynakubeName}, &dynakube); !k8serrors.IsNotFound(err) {
			continue
		}

		if err := checker.access.DeleteDynakube(ctx, dynakubeName); err != nil {
			return err
		}

		tenantUUID := dynakubes[dynakubeName]
		pruned = append(pruned, tenantUUID+"|"+dynakubeName)
	}

	log.Info("CSI tenants database is corrected for missing dynakubes (tenant|dynakube)", "prunedRows", pruned)

	return nil
}

func (checker *CorrectnessChecker) copyCodeModulesFromDeprecatedBin(ctx context.Context) error {
	dynakubes, err := checker.access.GetAllDynakubes(ctx)
	if err != nil {
		return err
	}

	moved := []string{}

	for _, dynakube := range dynakubes {
		if dynakube.TenantUUID == "" || dynakube.LatestVersion == "" {
			continue
		}

		deprecatedBin := checker.path.AgentBinaryDirForVersion(dynakube.TenantUUID, dynakube.LatestVersion)
		currentBin := checker.path.AgentSharedBinaryDirForAgent(dynakube.LatestVersion)

		linked, err := checker.safelyLinkCodeModule(deprecatedBin, currentBin)
		if err != nil {
			return err
		}

		if linked {
			moved = append(moved, dynakube.TenantUUID+"|"+dynakube.LatestVersion)
		}
	}

	log.Info("CSI filesystem corrected, linked deprecated agent binary to current location (tenant|version-bin)", "movedBins", moved)

	return nil
}

func (checker *CorrectnessChecker) safelyLinkCodeModule(deprecatedBin, currentBin string) (bool, error) {
	if folderExists(checker.fs, deprecatedBin) && !folderExists(checker.fs, currentBin) {
		log.Info("linking codemodule from deprecated location", "path", deprecatedBin)
		// MemMapFs (used for testing) doesn't comply with the Linker interface
		linker, ok := checker.fs.(afero.Linker)
		if !ok {
			log.Info("symlinking not possible", "path", deprecatedBin)

			return false, nil
		}

		err := checker.fs.MkdirAll(filepath.Dir(currentBin), 0755)
		if err != nil {
			log.Info("failed to create parent dir for new path", "path", currentBin)

			return false, errors.WithStack(err)
		}

		log.Info("creating symlink", "from", deprecatedBin, "to", currentBin)

		if err := linker.SymlinkIfPossible(deprecatedBin, currentBin); err != nil {
			log.Info("symlinking failed", "path", deprecatedBin)

			return false, errors.WithStack(err)
		}

		return true, nil
	}

	return false, nil
}

func folderExists(fs afero.Fs, filename string) bool {
	info, err := fs.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return info.IsDir()
}
