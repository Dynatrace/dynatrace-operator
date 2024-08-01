package csigc

import (
	"context"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	mount "k8s.io/mount-utils"
)

func (gc *CSIGarbageCollector) runBinaryGarbageCollection(ctx context.Context, tenantUUID string) error {
	binDirs, err := gc.getSharedBinDirs()
	if err != nil {
		return err
	}

	oldBinDirs, err := gc.getTenantBinDirs(tenantUUID)
	if err != nil {
		return err
	}

	binDirs = append(binDirs, oldBinDirs...)

	binsToDelete, err := gc.collectUnusedAgentBins(ctx, binDirs, tenantUUID)
	if err != nil {
		return err
	}

	if len(binsToDelete) == 0 {
		log.Info("no shared binary dirs to delete on the node")

		return nil
	}

	return gc.deleteBinDirs(binsToDelete)
}

func (gc *CSIGarbageCollector) collectUnusedAgentBins(ctx context.Context, imageDirs []os.FileInfo, tenantUUID string) ([]string, error) {
	var toDelete []string

	usedAgentVersions, err := gc.db.GetLatestVersions(ctx)
	if err != nil {
		log.Info("failed to get the used image versions")

		return nil, err
	}

	usedAgentDigest, err := gc.db.GetUsedImageDigests(ctx)
	if err != nil {
		log.Info("failed to get the used image digests")

		return nil, err
	}

	mountedAgentBins, err := getRelevantOverlayMounts(gc.mounter, []string{gc.path.AgentBinaryDir(tenantUUID), gc.path.AgentSharedBinaryDirBase()})
	if err != nil {
		log.Info("failed to get all mounted versions")

		return nil, err
	}

	for _, imageDir := range imageDirs {
		agentBin := imageDir.Name()
		sharedPath := gc.path.AgentSharedBinaryDirForAgent(agentBin)
		tenantPath := gc.path.AgentBinaryDirForVersion(tenantUUID, agentBin)

		switch {
		case usedAgentVersions[agentBin]: // versions that may not be used, but a dynakube references it
			continue
		case usedAgentDigest[agentBin]: // images that may not be used, but a dynakube references it
			continue
		}

		if !mountedAgentBins[sharedPath] { // based on mount, active shared codemodule mounts
			toDelete = append(toDelete, sharedPath)
		}

		if !mountedAgentBins[tenantPath] { // based on mount, active tenant codemodule mounts
			toDelete = append(toDelete, tenantPath)
		}
	}

	return toDelete, nil
}

func (gc *CSIGarbageCollector) deleteBinDirs(imageDirs []string) error {
	for _, dir := range imageDirs {
		err := gc.fs.RemoveAll(dir)
		if err != nil {
			log.Info("failed to delete codemodule bin dir", "dir", dir)

			return errors.WithStack(err)
		}
	}

	return nil
}

func (gc *CSIGarbageCollector) getTenantBinDirs(tenantUUID string) ([]os.FileInfo, error) {
	binPath := gc.path.AgentBinaryDir(tenantUUID)

	binDirs, err := afero.Afero{Fs: gc.fs}.ReadDir(binPath)
	if os.IsNotExist(err) {
		log.Info("no codemodule versions stored in deprecated path", "path", binPath)

		return nil, nil
	} else if err != nil {
		log.Info("failed to read codemodule versions stored in deprecated path", "path", binPath)

		return nil, errors.WithStack(err)
	}

	return binDirs, nil
}

func (gc *CSIGarbageCollector) getSharedBinDirs() ([]os.FileInfo, error) {
	sharedPath := gc.path.AgentSharedBinaryDirBase()

	imageDirs, err := afero.Afero{Fs: gc.fs}.ReadDir(sharedPath)
	if os.IsNotExist(err) {
		log.Info("no shared codemodules stored ", "path", sharedPath)

		return nil, nil
	}

	if err != nil {
		log.Info("failed to read shared image directory", "path", sharedPath)

		return nil, errors.WithStack(err)
	}

	return imageDirs, nil
}

func getRelevantOverlayMounts(mounter mount.Interface, baseFolders []string) (map[string]bool, error) {
	mountPoints, err := mounter.List()
	if err != nil {
		log.Error(err, "failed to list all mount points")

		return nil, err
	}

	relevantMounts := map[string]bool{}

	for _, mountPoint := range mountPoints {
		if mountPoint.Device == "overlay" {
			for _, opt := range mountPoint.Opts {
				for _, baseFolder := range baseFolders {
					if strings.HasPrefix(opt, "lowerdir="+baseFolder) {
						split := strings.Split(opt, "=")
						relevantMounts[split[1]] = true

						break
					}
				}
			}
		}
	}

	return relevantMounts, nil
}
