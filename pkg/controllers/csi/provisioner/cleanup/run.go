package cleanup

import (
	"context"
	"os"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"k8s.io/mount-utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cleaner struct {
	apiReader client.Reader
	mounter   mount.Interface
	path      metadata.PathResolver
}

// fsState collects all the "top-level" folders we care about and categorizes them
type fsState struct {
	// deprecatedDks are the dynakube-dirs under /data that have a /run directory, which used to contain the app-mounts, these directories are named after the tenantUUID
	deprecatedDks []string
	// binDks are dynakube-dirs that have a "latest-symlink" pointing at a codemodule binary
	binDks []string
	// hostDks are dynakube-dirs that contain the folder that was mounted to Host OneAgents
	hostDks []string
}

func New(apiReader client.Reader, path metadata.PathResolver, mounter mount.Interface) *Cleaner {
	return &Cleaner{
		apiReader: apiReader,
		path:      path,
		mounter:   mounter,
	}
}

// Run will only execute the cleanup logic if enough time has passed from the previous run, to not overload the IO of the node
func (c *Cleaner) Run(ctx context.Context) error {
	tickerResetFunc := checkTicker()
	if tickerResetFunc == nil {
		return nil
	}
	defer tickerResetFunc()

	return c.run(ctx)
}

// InstantRun will always execute the cleanup logic ignoring the time passed from previous run
func (c *Cleaner) InstantRun(ctx context.Context) error {
	defer resetTickerAfterDelete()

	return c.run(ctx)
}

func (c *Cleaner) run(ctx context.Context) error {
	fsState, err := c.getFilesystemState()
	if err != nil {
		return err
	}

	c.removeDeprecatedMounts(fsState)

	dks, err := metadata.GetRelevantDynaKubes(ctx, c.apiReader)
	if err != nil {
		log.Info("failed to list available dynakubes, skipping cleanup")

		return err
	}

	c.removeHostMounts(dks, fsState)
	c.removeUnusedBinaries(dks, fsState)

	return nil
}

func (c *Cleaner) getFilesystemState() (fsState fsState, err error) { //nolint:revive
	rootSubDirs, err := os.ReadDir(c.path.RootDir)
	if err != nil {
		log.Info("failed to list the contents of the root directory of the csi-provisioner", "rootDir", c.path.RootDir)

		return fsState, err
	}

	var unknownDirs []string

	defer func() {
		for _, unknown := range unknownDirs {
			log.Info("removing unknown path", "path", unknown)
			_ = os.RemoveAll(unknown)
		}
	}()

	for _, fileInfo := range rootSubDirs {
		if !fileInfo.IsDir() ||
			fileInfo.Name() == dtcsi.SharedAppMountsDir ||
			fileInfo.Name() == dtcsi.SharedJobWorkDir ||
			fileInfo.Name() == dtcsi.SharedDynaKubesDir ||
			fileInfo.Name() == dtcsi.SharedAgentBinDir {
			continue
		}

		var deprecatedExists, hostExists bool

		_, err := os.Stat(c.path.AgentRunDir(fileInfo.Name()))
		if err == nil {
			deprecatedExists = true

			fsState.deprecatedDks = append(fsState.deprecatedDks, fileInfo.Name())
		}

		_, err = os.Stat(c.path.OldOsAgentDir(fileInfo.Name()))
		if err == nil {
			hostExists = true

			fsState.hostDks = append(fsState.hostDks, fileInfo.Name())
		}

		if !deprecatedExists && !hostExists {
			unknownDirs = append(unknownDirs, c.path.Base(fileInfo.Name()))
		}
	}

	dkDirs, err := os.ReadDir(c.path.DynaKubesBaseDir())
	if os.IsNotExist(err) {
		return fsState, nil
	} else if err != nil {
		log.Info("failed to list the contents of the dynakube directory of the csi-provisioner", "dynakubes folder", c.path.DynaKubesBaseDir())

		return fsState, err
	}

	for _, fileInfo := range dkDirs {
		if !fileInfo.IsDir() {
			continue
		}

		var binExists, hostExists bool

		_, err := os.Stat(c.path.LatestAgentBinaryForDynaKube(fileInfo.Name()))
		if err == nil {
			binExists = true

			fsState.binDks = append(fsState.binDks, fileInfo.Name())
		}

		_, err = os.Stat(c.path.OsAgentDir(fileInfo.Name()))
		if err == nil {
			hostExists = true

			fsState.hostDks = append(fsState.hostDks, fileInfo.Name())
		}

		if !binExists && !hostExists {
			unknownDirs = append(unknownDirs, c.path.DynaKubeDir(fileInfo.Name()))
		}
	}

	return fsState, nil
}

// safeAddRelevantPath follows the symlink that is provided in the `path` param and adds the actual path to the provided map
// It checks for the existence of the path and verifies if it is a symlink.
// Trying to follow a path that is not a symlink will case an error.
// Should be used for paths that are "maybe" symlinks, more expensive then its addRelevantPath.
func (c *Cleaner) safeAddRelevantPath(path string, relevantPaths map[string]bool) {
	fInfo, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error(err, "failed to check if host mount directory is a symlink")
		}

		return
	}

	if fInfo.Mode() != os.ModeSymlink {
		relevantPaths[path] = true

		return
	}

	c.addRelevantPath(path, relevantPaths)
}

// addRelevantPath follows the symlink that is provided in the `path` param and adds the actual path to the provided map
// does no checking for the existence of the path and does not verify if it is a symlink.
// Should be used for paths that are 100% to be symlinks to save on IO.
func (c *Cleaner) addRelevantPath(path string, relevantPaths map[string]bool) {
	actualPath, err := os.Readlink(path)
	if err != nil {
		log.Error(err, "failed to follow symlink", "path", path)

		return
	}

	relevantPaths[actualPath] = true
}
