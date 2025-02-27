package cleanup

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/mount-utils"
)

func TestRemoveUnusedBinaries(t *testing.T) {
	// Not possible to test, as parts of it rely on turning symlinks into actual paths, and clearing up according to these paths.
	// each individual part is testable
	t.SkipNow()
}

func TestRemoveOldSharedBinaries(t *testing.T) {
	t.Run("empty fs -> no panic", func(t *testing.T) {
		cleaner := createCleaner(t)

		keptBins := map[string]bool{}

		cleaner.removeOldSharedBinaries(keptBins)
	})
	t.Run("empty shared dir -> no panic", func(t *testing.T) {
		cleaner := createCleaner(t)
		cleaner.fs.MkdirAll(cleaner.path.AgentSharedBinaryDirBase(), os.ModePerm)

		keptBins := map[string]bool{}

		cleaner.removeOldSharedBinaries(keptBins)
	})

	t.Run("empty keptBins -> remove all", func(t *testing.T) {
		cleaner := createCleaner(t)
		cleaner.fs.MkdirAll(cleaner.path.AgentSharedBinaryDirBase(), os.ModePerm)

		keptBins := map[string]bool{}
		agentVersions := []string{"test1", "test2"}

		for _, folder := range agentVersions {
			cleaner.createSharedBinDir(t, folder)

			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(folder)
			exists, _ := cleaner.fs.Exists(expectedDir)
			require.True(t, exists)
		}

		cleaner.removeOldSharedBinaries(keptBins)

		for _, folder := range agentVersions {
			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(folder)
			exists, _ := cleaner.fs.Exists(expectedDir)
			require.False(t, exists)
		}
	})

	t.Run("keptBins set -> only remove orphans", func(t *testing.T) {
		cleaner := createCleaner(t)
		cleaner.fs.MkdirAll(cleaner.path.AgentSharedBinaryDirBase(), os.ModePerm)

		keptBins := map[string]bool{
			cleaner.path.AgentSharedBinaryDirForAgent("test1"): true,
			cleaner.path.AgentSharedBinaryDirForAgent("test2"): true,
		}
		agentVersions := []string{"test1", "test2"}
		orphans := []string{"o1", "o2"}

		for _, version := range append(agentVersions, orphans...) {
			cleaner.createSharedBinDir(t, version)

			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(version)
			exists, _ := cleaner.fs.Exists(expectedDir)
			require.True(t, exists)
		}

		cleaner.removeOldSharedBinaries(keptBins)

		for _, folder := range agentVersions {
			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(folder)
			exists, _ := cleaner.fs.Exists(expectedDir)
			require.True(t, exists)
		}

		for _, folder := range orphans {
			expectedDir := cleaner.path.AgentSharedBinaryDirForAgent(folder)
			exists, _ := cleaner.fs.Exists(expectedDir)
			require.False(t, exists)
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
		cleaner.path.RootDir = "special"
		expectedPath := cleaner.path.RootDir + "/something"
		expectedLowerDir := expectedPath + "/else"

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

		relevantBins := cleaner.collectRelevantLatestBins([]dynakube.DynaKube{
			createAppMonDk(t, "appmon", "url"),
		})

		require.NotEmpty(t, relevantBins)
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
			exists, _ := cleaner.fs.Exists(expectedDir)
			require.True(t, exists)
		}

		cleaner.removeOldBinarySymlinks(dks, fsState{
			binDks: binDirs,
		})

		for _, folder := range binDirs {
			exists, _ := cleaner.fs.Exists(cleaner.path.LatestAgentBinaryForDynaKube(folder))
			require.False(t, exists)
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
			exists, _ := cleaner.fs.Exists(expectedDir)
			require.True(t, exists)
		}

		cleaner.removeOldBinarySymlinks(dks, fsState{
			binDks: binDirs,
		})

		for _, folder := range binDirs[:2] {
			exists, _ := cleaner.fs.Exists(cleaner.path.LatestAgentBinaryForDynaKube(folder))
			require.True(t, exists)
		}

		for _, folder := range binDirs[2:] {
			exists, _ := cleaner.fs.Exists(cleaner.path.LatestAgentBinaryForDynaKube(folder))
			require.False(t, exists)
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
			exists, _ := cleaner.fs.Exists(expectedDir)
			require.True(t, exists)
		}

		cleaner.removeOldBinarySymlinks(dks, fsState{
			deprecatedDks: binDirs,
		})

		for _, folder := range binDirs[:2] {
			exists, _ := cleaner.fs.Exists(cleaner.path.LatestAgentBinaryForDynaKube(folder))
			require.True(t, exists)
		}

		for _, folder := range binDirs[2:] {
			exists, _ := cleaner.fs.Exists(cleaner.path.LatestAgentBinaryForDynaKube(folder))
			require.False(t, exists)
		}
	})
}

func mockMountPoints(t *testing.T, cleaner *Cleaner, mountPoints ...mount.MountPoint) {
	t.Helper()

	cleaner.mounter = mount.NewFakeMounter(mountPoints)
}

func createAppMonDk(t *testing.T, name, apiUrl string) dynakube.DynaKube {
	t.Helper()

	dk := createBaseDk(t, name, apiUrl)
	dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}

	return dk
}

func createBaseDk(t *testing.T, name, apiUrl string) dynakube.DynaKube {
	t.Helper()

	return dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: apiUrl,
		},
	}
}

func (c *Cleaner) createSharedBinDir(t *testing.T, version string) {
	t.Helper()

	binDir := c.path.AgentSharedBinaryDirForAgent(version)
	err := c.fs.MkdirAll(binDir, os.ModePerm)
	require.NoError(t, err)
}
