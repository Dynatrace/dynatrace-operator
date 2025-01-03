package cleanup

import (
	"strings"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"golang.org/x/exp/maps"
	"k8s.io/mount-utils"
)

type Cleaner struct {
	fs      afero.Afero
	mounter mount.Interface
	path    metadata.PathResolver
}

func New(fs afero.Afero, path metadata.PathResolver) *Cleaner {
	return &Cleaner{
		fs:      fs,
		path:    path,
		mounter: mount.New(""),
	}
}

func (c Cleaner) Run() error {
	tickerResetFunc := checkTicker()
	if tickerResetFunc == nil {
		return nil
	}
	defer tickerResetFunc()

	return c.run()
}

func (c Cleaner) run() error {
	rootSubDirs, err := c.fs.ReadDir(c.path.RootDir)
	if err != nil {
		return err
	}

	var tenantDirsWithDeprecatedFolders []string

	var relevantBinDirs []string

	for _, fileInfo := range rootSubDirs {
		if !fileInfo.IsDir() ||
			fileInfo.Name() == dtcsi.SharedAppMountsDir ||
			fileInfo.Name() == dtcsi.SharedAgentBinDir {
			continue
		}

		_, err := c.fs.Stat(c.path.AgentRunDir(fileInfo.Name()))
		if err == nil {
			tenantDirsWithDeprecatedFolders = append(tenantDirsWithDeprecatedFolders, fileInfo.Name())

			continue
		}

		latestBinDir := c.path.LatestAgentBinaryForDynaKube(fileInfo.Name())

		_, err = c.fs.Stat(latestBinDir)
		if err == nil {
			continue
		}

		linker, ok := c.fs.Fs.(afero.LinkReader)
		if ok {
			actualPath, err := linker.ReadlinkIfPossible(latestBinDir)
			if err != nil {
				log.Error(err, "failed to follow symlink", "path", latestBinDir)

				continue
			}

			relevantBinDirs = append(relevantBinDirs, actualPath)
		}
	}

	c.removeDeprecatedMounts(tenantDirsWithDeprecatedFolders)

	err = c.removeUnusedBinaries(relevantBinDirs)
	if err != nil {
		return err
	}

	return nil
}

func (c Cleaner) removeDeprecatedMounts(tenantNames []string) {
	for _, tenant := range tenantNames {
		runDir := c.path.AgentRunDir(tenant)

		volumeDirs, err := c.fs.ReadDir(runDir)
		if err != nil {
			log.Info("couldn't list volume dirs", "path", runDir)

			continue
		}

		for _, volumeDir := range volumeDirs {
			mappedDir := c.path.OverlayMappedDir(tenant, volumeDir.Name())

			isEmpty, _ := c.fs.IsEmpty(mappedDir)
			if isEmpty {
				volumeDirPath := c.path.AgentRunDirForVolume(tenant, volumeDir.Name())

				err := c.fs.RemoveAll(volumeDirPath)
				if err == nil {
					log.Info("removed unused volume", "path", volumeDirPath)
				}
			}
		}

		tenantDir := c.path.DynaKubeDir(tenant)

		isEmpty, _ := c.fs.IsEmpty(tenantDir)
		if isEmpty {
			err := c.fs.RemoveAll(tenantDir)
			if err == nil {
				log.Info("removed empty old tenant folder", "path", tenantDir)
			}
		}
	}
}

func (c Cleaner) removeUnusedBinaries(latestBins []string) error {
	overlays, err := metadata.GetRelevantOverlayMounts(c.mounter, c.path.RootDir)
	if err != nil {
		log.Info("failed to list active overlay mounts, skipping unused binaries cleanup")

		return err
	}

	keptBins := map[string]bool{}
	for _, overlay := range overlays {
		keptBins[overlay.LowerDir] = true
	}

	log.Info("binaries to keep because they are still mounted", "paths", strings.Join(maps.Keys(keptBins), ","))

	for _, latest := range latestBins {
		keptBins[latest] = true
	}

	log.Info("binaries to keep because they are the latest", "paths", strings.Join(latestBins, ","))

	sharedBins, err := c.fs.ReadDir(c.path.AgentSharedBinaryDirBase())
	if err != nil {
		log.Info("failed to list the shared binaries directory, skipping unused binaries cleanup")

		return err
	}

	for _, dir := range sharedBins {
		sharedBinPath := c.path.AgentSharedBinaryDirForAgent(dir.Name())

		_, ok := keptBins[sharedBinPath]
		if !ok {
			err := c.fs.RemoveAll(sharedBinPath)
			if err != nil {
				log.Error(err, "failed to remove shared binary", "path", sharedBinPath)

				continue
			}

			log.Info("removed old shared binary", "path", sharedBinPath)
		}
	}

	return nil
}
