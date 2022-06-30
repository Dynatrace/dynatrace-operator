package csigc

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (gc *CSIGarbageCollector) runSharedImagesGarbageCollection() error {
	imageDirs, err := gc.getSharedImageDirs()
	if err != nil {
		return err
	}
	if len(imageDirs) == 0 {
		log.Info("no shared image dirs on node")
		return nil
	}

	imagesToDelete, err := gc.collectUnusedImageDirs(imageDirs)
	if err != nil {
		return err
	}
	if len(imagesToDelete) == 0 {
		log.Info("no shared image dirs to delete on the node")
		return nil
	}

	return deleteImageDirs(gc.fs, imagesToDelete)
}

func (gc *CSIGarbageCollector) getSharedImageDirs() ([]os.FileInfo, error) {
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

func (gc *CSIGarbageCollector) collectUnusedImageDirs(imageDirs []os.FileInfo) ([]string, error) {
	var toDelete []string
	usedImageDigests, err := gc.getUsedImageDigests()
	if err != nil {
		log.Info("failed to get the used image digests")
		return nil, err
	}
	for _, imageDir := range imageDirs {
		if !imageDir.IsDir() {
			continue
		}
		imageDigest := imageDir.Name()
		if !usedImageDigests[imageDigest] {
			toDelete = append(toDelete, gc.path.AgentSharedBinaryDirForImage(imageDigest))
		}
	}
	return toDelete, nil
}

func (gc *CSIGarbageCollector) getUsedImageDigests() (map[string]bool, error) {
	usedImageDigests, err := gc.db.GetUsedImageDigests()
	if err != nil {
		log.Info("failed to get the used image digests")
		return nil, err
	}

	// If a shared image was used during mount, the version of a Volume is the imageDigest.
	// A Volume can still reference versions that are not imageDigests.
	// However, this shouldn't cause issues as those versions don't matter in this context.
	usedVersions, err := gc.db.GetAllUsedVersions()
	if err != nil {
		log.Info("failed to get all used versions")
		return nil, err
	}
	for version := range usedVersions {
		usedImageDigests[version] = true
	}
	return usedImageDigests, nil
}

func deleteImageDirs(fs afero.Fs, imageDirs []string) error {
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
