package cleanup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/mount-utils"
)

func TestRemoveUnusedBinaries(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		cleaner := createCleaner(t)

		// Setup latest bin -> should NOT be removed
		dk := createAppMonDk(t, "appmon", "url")
		relevantBin := cleaner.path.AgentSharedBinaryDirForAgent("1.2.3")
		require.NoError(t, os.MkdirAll(relevantBin, os.ModePerm))
		require.NoError(t, os.MkdirAll(cleaner.path.DynaKubeDir(dk.Name), os.ModePerm))
		require.NoError(t, os.Symlink(relevantBin, cleaner.path.LatestAgentBinaryForDynaKube(dk.Name)))

		// Setup still mounted bin -> should NOT be removed
		expectedPath := cleaner.path.AppMountForID("example")
		stillMountedBin := cleaner.path.AgentSharedBinaryDirForAgent("1.1.1")
		require.NoError(t, os.MkdirAll(expectedPath, os.ModePerm))
		require.NoError(t, os.MkdirAll(stillMountedBin, os.ModePerm))
		relevantMountPoint := mount.MountPoint{
			Device: "overlay",
			Path:   expectedPath,
			Type:   "overlay",
			Opts: []string{
				"lowerdir=" + stillMountedBin,
				"upperdir=...",
				"workdir=...",
			},
		}
		mockMountPoints(t, cleaner, relevantMountPoint)

		// Setup unused bins -> should be removed
		unusedVersions := []string{"1.0.0", "1.0.1", "1.1.0"}
		for _, version := range unusedVersions {
			require.NoError(t, os.MkdirAll(cleaner.path.AgentSharedBinaryDirForAgent(version), os.ModePerm))
		}

		// Setup fsState, with old dks -> unused dks should be removed
		state := fsState{
			binDks: []string{
				dk.Name, "dk1", "dk2", "dk3", // dk.Name is the only one the will remain
			},
			deprecatedDks: []string{
				"tenant1", "tenant2", "tenant3",
			},
		}
		for _, dkName := range state.binDks {
			require.NoError(t, os.MkdirAll(cleaner.path.LatestAgentBinaryForDynaKube(dkName), os.ModePerm))
		}
		for _, tenantName := range state.deprecatedDks {
			require.NoError(t, os.MkdirAll(cleaner.path.LatestAgentBinaryForDynaKube(tenantName), os.ModePerm))
		}

		cleaner.removeUnusedBinaries([]dynakube.DynaKube{dk}, state)

		for _, version := range unusedVersions {
			assert.NoDirExists(t, cleaner.path.AgentSharedBinaryDirForAgent(version))
		}

		// Exists because there is a dk for it
		assert.DirExists(t, cleaner.path.AgentSharedBinaryDirForAgent("1.2.3"))

		// Exists because there is a mount still using it
		assert.DirExists(t, cleaner.path.AgentSharedBinaryDirForAgent("1.1.1"))

		for _, dkName := range state.binDks {
			if dkName == dk.Name {
				// Exists because there is a dk for it
				assert.FileExists(t, cleaner.path.LatestAgentBinaryForDynaKube(dkName))

				continue
			}
			assert.NoFileExists(t, cleaner.path.LatestAgentBinaryForDynaKube(dkName))
		}
		for _, tenantName := range state.deprecatedDks {
			assert.NoDirExists(t, cleaner.path.LatestAgentBinaryForDynaKube(tenantName))
		}
	})
}

func TestRemoveOldSharedBinaries(t *testing.T) {
	t.Run("empty fs -> no panic", func(t *testing.T) {
		cleaner := createCleaner(t)

		keptBins := map[string]bool{}

		cleaner.removeOldSharedBinaries(keptBins)
	})
	t.Run("empty shared dir -> no panic", func(t *testing.T) {
		cleaner := createCleaner(t)
		os.MkdirAll(cleaner.path.AgentSharedBinaryDirBase(), os.ModePerm)

		keptBins := map[string]bool{}

		cleaner.removeOldSharedBinaries(keptBins)
	})

	t.Run("empty keptBins -> remove all", func(t *testing.T) {
		cleaner := createCleaner(t)
		os.MkdirAll(cleaner.path.AgentSharedBinaryDirBase(), os.ModePerm)

		keptBins := map[string]bool{}
		agentVersions := []string{"test1", "test2"}

		for _, folder := range agentVersions {
			cleaner.createSharedBinDir(t, folder)

			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(folder)
			assert.DirExists(t, expectedDir)
		}

		cleaner.removeOldSharedBinaries(keptBins)

		for _, folder := range agentVersions {
			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(folder)
			assert.NoDirExists(t, expectedDir)
		}
	})

	t.Run("keptBins set -> only remove orphans", func(t *testing.T) {
		cleaner := createCleaner(t)
		os.MkdirAll(cleaner.path.AgentSharedBinaryDirBase(), os.ModePerm)

		keptBins := map[string]bool{
			cleaner.path.AgentSharedBinaryDirForAgent("test1"): true,
			cleaner.path.AgentSharedBinaryDirForAgent("test2"): true,
		}
		agentVersions := []string{"test1", "test2"}
		orphans := []string{"o1", "o2"}

		for _, version := range append(agentVersions, orphans...) {
			cleaner.createSharedBinDir(t, version)

			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(version)
			assert.DirExists(t, expectedDir)
		}

		cleaner.removeOldSharedBinaries(keptBins)

		for _, folder := range agentVersions {
			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(folder)
			assert.DirExists(t, expectedDir)
		}

		for _, folder := range orphans {
			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(folder)
			assert.NoDirExists(t, expectedDir)
		}
	})
}

func TestCollectStillMountedBins(t *testing.T) {
	t.Run("0 mounts -> empty", func(t *testing.T) {
		cleaner := createCleaner(t)

		relevantBins, err := cleaner.collectStillMountedBins()

		require.NoError(t, err)
		require.Empty(t, relevantBins)
	})
	t.Run("get mounted bins", func(t *testing.T) {
		cleaner := createCleaner(t)
		cleaner.path.RootDir = filepath.Join(cleaner.path.RootDir, "special")
		expectedPath := filepath.Join(cleaner.path.RootDir, "something")
		expectedLowerDir := filepath.Join(cleaner.path.RootDir, "else")

		relevantMountPoint := mount.MountPoint{
			Device: "overlay",
			Path:   expectedPath,
			Type:   "overlay",
			Opts: []string{
				"lowerdir=" + expectedLowerDir,
				"upperdir=...",
				"workdir=...",
			},
		}

		mockMountPoints(t, cleaner, relevantMountPoint)

		relevantBins, err := cleaner.collectStillMountedBins()

		require.NoError(t, err)
		require.Len(t, relevantBins, 1)
		assert.True(t, relevantBins[expectedLowerDir])
	})
}

func TestCollectRelevantLatestBins(t *testing.T) {
	t.Run("no dk -> do nothing", func(t *testing.T) {
		cleaner := createCleaner(t)

		relevantBins := cleaner.collectRelevantLatestBins([]dynakube.DynaKube{})

		require.Empty(t, relevantBins)
	})
	t.Run("no relevant dk -> do nothing", func(t *testing.T) {
		cleaner := createCleaner(t)

		relevantBins := cleaner.collectRelevantLatestBins([]dynakube.DynaKube{
			createHostMonDk(t, "hostmon", "url"),
		})

		require.Empty(t, relevantBins)
	})

	t.Run("relevant dk -> try to resolve symlink", func(t *testing.T) {
		cleaner := createCleaner(t)
		dk := createAppMonDk(t, "appmon", "url")
		relevantBin := cleaner.path.AgentSharedBinaryDirForAgent("1.2.3")
		require.NoError(t, os.MkdirAll(relevantBin, os.ModePerm))
		require.NoError(t, os.MkdirAll(cleaner.path.DynaKubeDir(dk.Name), os.ModePerm))
		require.NoError(t, os.Symlink(relevantBin, cleaner.path.LatestAgentBinaryForDynaKube(dk.Name)))

		relevantBins := cleaner.collectRelevantLatestBins([]dynakube.DynaKube{
			dk,
		})

		require.NotEmpty(t, relevantBins)
		assert.Contains(t, relevantBins, relevantBin)
	})
}

func TestRemoveOldBinarySymlinks(t *testing.T) {
	t.Run("no dk -> remove everything", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{}

		binDirs := []string{"test1", "test2"}

		for _, folder := range binDirs {
			cleaner.createBinDirs(t, folder)

			expectedDir := cleaner.path.LatestAgentBinaryForDynaKube(folder)
			assert.DirExists(t, expectedDir)
		}

		cleaner.removeOldBinarySymlinks(dks, fsState{
			binDks: binDirs,
		})

		for _, folder := range binDirs {
			assert.NoDirExists(t, cleaner.path.LatestAgentBinaryForDynaKube(folder))
		}
	})

	t.Run("dk -> don't remove", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{
			createCloudNativeDk(t, "cloudnative", "-"),
			createAppMonDk(t, "appmon", "-"),
		}

		binDirs := []string{dks[0].Name, dks[1].Name, "test1", "test2"}

		for _, folder := range binDirs {
			cleaner.createBinDirs(t, folder)

			expectedDir := cleaner.path.LatestAgentBinaryForDynaKube(folder)
			assert.DirExists(t, expectedDir)
		}

		cleaner.removeOldBinarySymlinks(dks, fsState{
			binDks: binDirs,
		})

		for _, folder := range binDirs[:2] {
			assert.DirExists(t, cleaner.path.LatestAgentBinaryForDynaKube(folder))
		}

		for _, folder := range binDirs[2:] {
			assert.NoDirExists(t, cleaner.path.LatestAgentBinaryForDynaKube(folder))
		}
	})

	t.Run("dk.Name == tenantUUID -> don't remove", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{
			createCloudNativeDk(t, "cloudnative", "-"),
			createAppMonDk(t, "appmon", "-"),
		}

		binDirs := []string{dks[0].Name, dks[1].Name, "test1", "test2"}

		for _, folder := range binDirs {
			cleaner.createBinDirs(t, folder)

			expectedDir := cleaner.path.LatestAgentBinaryForDynaKube(folder)
			assert.DirExists(t, expectedDir)
		}

		cleaner.removeOldBinarySymlinks(dks, fsState{
			deprecatedDks: binDirs,
		})

		for _, folder := range binDirs[:2] {
			assert.DirExists(t, cleaner.path.LatestAgentBinaryForDynaKube(folder))
		}

		for _, folder := range binDirs[2:] {
			assert.NoDirExists(t, cleaner.path.LatestAgentBinaryForDynaKube(folder))
		}
	})
}

func mockMountPoints(t *testing.T, cleaner *Cleaner, mountPoints ...mount.MountPoint) {
	t.Helper()

	cleaner.mounter = mount.NewFakeMounter(mountPoints)
}

func createAppMonDk(t *testing.T, name, apiURL string) dynakube.DynaKube {
	t.Helper()

	dk := createBaseDk(t, name, apiURL)
	dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}

	return dk
}

func createBaseDk(t *testing.T, name, apiURL string) dynakube.DynaKube {
	t.Helper()

	return dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: apiURL,
		},
	}
}

func (c *Cleaner) createSharedBinDir(t *testing.T, version string) {
	t.Helper()

	binDir := c.path.AgentSharedBinaryDirForAgent(version)
	err := os.MkdirAll(binDir, os.ModePerm)
	require.NoError(t, err)
}
