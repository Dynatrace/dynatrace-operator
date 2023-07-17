package csigc

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (gc *CSIGarbageCollector) runSharedBinaryGarbageCollection(ctx context.Context) error {
	imageDirs, err := gc.getSharedBinDirs()
	if err != nil {
		return err
	}
	if len(imageDirs) == 0 {
		log.Info("no shared binary dirs on node")
		return nil
	}

	binsToDelete, err := gc.collectUnusedAgentBins(ctx, imageDirs)
	if err != nil {
		return err
	}
	if len(binsToDelete) == 0 {
		log.Info("no shared binary dirs to delete on the node")
		return nil
	}

	return deleteSharedBinDirs(gc.fs, binsToDelete)
}

func (gc *CSIGarbageCollector) getSharedBinDirs() ([]os.FileInfo, error) {
	imageDirs, err := afero.Afero{Fs: gc.fs}.ReadDir(gc.path.AgentSharedBinaryDirBase())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		log.Info("failed to read shared image directory")
		return nil, errors.WithStack(err)
	}
	return imageDirs, nil
}

func (gc *CSIGarbageCollector) collectUnusedAgentBins(ctx context.Context, imageDirs []os.FileInfo) ([]string, error) {
	var toDelete []string
	setAgentVersions, err := gc.db.GetLatestVersions(ctx)
	if err != nil {
		log.Info("failed to get the set image versions")
		return nil, err
	}
	setAgentImages, err := gc.db.GetUsedImageDigests(ctx)
	if err != nil {
		log.Info("failed to get the set image digests")
		return nil, err
	}

	// If a shared image was used during mount, the version of a Volume is the imageDigest.
	// A Volume can still reference versions that are not imageDigests.
	// However, this shouldn't cause issues as those versions don't matter in this context.
	mountedAgentBins, err := gc.db.GetAllUsedVersions(ctx)
	if err != nil {
		log.Info("failed to get all mounted versions")
		return nil, err
	}
	for _, imageDir := range imageDirs {
		if !imageDir.IsDir() {
			continue
		}
		agentBin := imageDir.Name()
		if !mountedAgentBins[agentBin] && !setAgentVersions[agentBin] && !setAgentImages[agentBin] {
			toDelete = append(toDelete, gc.path.AgentSharedBinaryDirForAgent(agentBin))
		}
	}
	return toDelete, nil
}

func deleteSharedBinDirs(fs afero.Fs, imageDirs []string) error {
	for _, dir := range imageDirs {
		log.Info("deleting shared image dir", "dir", dir)
		err := fs.RemoveAll(dir)
		if err != nil {
			log.Info("failed to delete image cache", "dir", dir)
			return errors.WithStack(err)
		}
	}
	return nil
}
