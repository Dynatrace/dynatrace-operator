package cleanup

import (
	"context"
	"path/filepath"
	"strings"

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

func (c Cleaner) Run(ctx context.Context) error {
	tickerResetFunc := checkTicker()
	if tickerResetFunc == nil {
		return nil
	}
	defer tickerResetFunc()

	rootSubDirs, err := c.fs.ReadDir(c.path.RootDir)
	if err != nil {
		return err
	}

	var tenantDirsWithDeprecatedFolders []string

	for _, fileInfo := range rootSubDirs {
		if !fileInfo.IsDir() || fileInfo.Name() == filepath.Base(c.path.AppMountsBaseDir()) {
			continue
		}

		_, err := c.fs.Stat(c.path.AgentRunDir(fileInfo.Name()))
		if err == nil {
			tenantDirsWithDeprecatedFolders = append(tenantDirsWithDeprecatedFolders, fileInfo.Name())

			continue
		}
	}
	c.removeDeprecatedMounts(tenantDirsWithDeprecatedFolders)
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
