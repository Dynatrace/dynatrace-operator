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
