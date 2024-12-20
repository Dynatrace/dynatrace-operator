package gc

import (
	"context"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
)

type Cleaner struct {
	fs   afero.Afero
	path metadata.PathResolver
}

func (c Cleaner) Run(ctx context.Context) error {
	rootSubDirs, err := c.fs.ReadDir(c.path.RootDir)
	if err != nil {
		return err
	}

	var deprecatedDirNames []string
	var relevantDirNames []string

	for _, fileInfo := range rootSubDirs {
		if !fileInfo.IsDir() || fileInfo.Name() == filepath.Base(c.path.AppMountsBaseDir()) {
			continue
		}

		_, err := c.fs.Stat(c.path.AgentRunDir(fileInfo.Name()))
		if err == nil {
			deprecatedDirNames = append(deprecatedDirNames, fileInfo.Name())

			continue
		}

		_, err = c.fs.Stat(c.path.LatestAgentBinaryForDynaKube(fileInfo.Name()))
		if err == nil {
			relevantDirNames = append(relevantDirNames, fileInfo.Name())

			continue
		}
	}

	err = c.removeDeprecatedMounts(deprecatedDirNames)
	if err != nil {
		return err
	}

	err = c.removeUnusedBinaries(relevantDirNames)
	if err != nil {
		return err
	}

	return nil
}

func (c Cleaner) removeDeprecatedMounts(tenantNames []string) error {
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

	return nil
}

func (c Cleaner) removeUnusedBinaries(dkNames []string) error {
	// TODO
	return nil
}
