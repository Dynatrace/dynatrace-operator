package cleanup

import (
	"fmt"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/mount-utils"
)

func TestRemoveHostMounts(t *testing.T) {
	tenantUUID1 := "tenant1"
	apiURL1 := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", tenantUUID1)

	tenantUUID2 := "tenant2"
	apiURL2 := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", tenantUUID2)

	t.Run("no dk -> no relevant dirs -> remove all", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{}
		hostFolders := []string{tenantUUID1, tenantUUID2, "random-name1", "random-name2"}

		for _, folder := range hostFolders {
			cleaner.createHostDirs(t, folder)

			assert.DirExists(t, cleaner.path.OsAgentDir(folder))
		}

		cleaner.removeHostMounts(dks, fsState{
			hostDks: hostFolders,
		})

		for _, folder := range hostFolders {
			assert.NoDirExists(t, cleaner.path.OsAgentDir(folder))
		}
	})

	t.Run("relevant dk -> remove only orphans", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{
			createHostMonDk(t, "hostmon", apiURL1),
			createCloudNativeDk(t, "cloudnative", apiURL2),
		}
		folders := []string{
			cleaner.path.OldOsAgentDir(tenantUUID1),
			cleaner.path.OsAgentDir(dks[0].Name),
			cleaner.path.OsAgentDir(dks[1].Name),
			cleaner.path.OldOsAgentDir("random-name1"),
			cleaner.path.OsAgentDir("random-name2"),
		}

		for _, folder := range folders {
			err := os.MkdirAll(folder, os.ModePerm)
			require.NoError(t, err)
		}

		cleaner.removeHostMounts(dks, fsState{
			hostDks: []string{dks[0].Name, dks[1].Name, tenantUUID1, "random-name1", "random-name2"},
		})

		for _, folder := range folders[:3] {
			assert.DirExists(t, folder)
		}

		for _, folder := range folders[3:] {
			assert.NoDirExists(t, folder)
		}
	})

	t.Run("don't remove mounted orphans", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{}
		hostFolders := []string{tenantUUID1, tenantUUID2}
		fakeMounter := mount.NewFakeMounter(nil)
		fakeMounter.MountCheckErrors = map[string]error{}

		for _, folder := range hostFolders {
			cleaner.createHostDirs(t, folder)

			expectedDir := cleaner.path.OsAgentDir(folder)
			assert.DirExists(t, expectedDir)

			fakeMounter.MountCheckErrors[expectedDir] = nil
		}

		cleaner.mounter = fakeMounter

		cleaner.removeHostMounts(dks, fsState{
			hostDks: hostFolders,
		})

		for _, folder := range hostFolders {
			assert.DirExists(t, cleaner.path.OsAgentDir(folder))
		}
	})
}

func TestCollectRelevantHostDirs(t *testing.T) {
	tenantUUID1 := "tenant1"
	apiURL1 := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", tenantUUID1)

	tenantUUID2 := "tenant2"
	apiURL2 := fmt.Sprintf("https://%s.dev.dynatracelabs.com/api", tenantUUID2)

	t.Run("no dk -> no relevant dirs", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{}

		relevantDirs := cleaner.collectRelevantHostDirs(dks)

		require.Empty(t, relevantDirs)
	})

	t.Run("not-relevant dk -> no relevant dirs", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{
			createAppMonDk(t, "appmon1", apiURL1),
			createAppMonDk(t, "appmon2", apiURL2),
		}

		relevantDirs := cleaner.collectRelevantHostDirs(dks)

		require.Empty(t, relevantDirs)
	})

	t.Run("relevant dks, but not existing -> current path always added", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{
			createHostMonDk(t, "hostmon", apiURL1),
			createCloudNativeDk(t, "cloudnative", apiURL2),
		}

		relevantDirs := cleaner.collectRelevantHostDirs(dks)

		require.NotEmpty(t, relevantDirs)
		require.Len(t, relevantDirs, 2)
		assert.Contains(t, relevantDirs, cleaner.path.OsAgentDir(dks[0].Name))
		assert.Contains(t, relevantDirs, cleaner.path.OsAgentDir(dks[1].Name))
		assert.NotContains(t, relevantDirs, cleaner.path.OsAgentDir(tenantUUID1))
		assert.NotContains(t, relevantDirs, cleaner.path.OsAgentDir(tenantUUID2))
	})

	t.Run("relevant dk -> relevant dirs, deprecated(tenantUUID) location dir included if exists", func(t *testing.T) {
		cleaner := createCleaner(t)
		dks := []dynakube.DynaKube{
			createHostMonDk(t, "hostmon", apiURL1),
			createCloudNativeDk(t, "cloudnative", apiURL2),
			createAppMonDk(t, "appmon", apiURL1),
		}

		cleaner.createDeprecatedHostDirs(t, tenantUUID1)
		cleaner.createHostDirs(t, dks[0].Name)

		relevantDirs := cleaner.collectRelevantHostDirs(dks)

		require.NotEmpty(t, relevantDirs)
		require.Len(t, relevantDirs, 3)
		assert.Contains(t, relevantDirs, cleaner.path.OsAgentDir(dks[0].Name))
		assert.Contains(t, relevantDirs, cleaner.path.OsAgentDir(dks[1].Name))
		assert.NotContains(t, relevantDirs, cleaner.path.OsAgentDir(dks[2].Name))
		assert.Contains(t, relevantDirs, cleaner.path.OldOsAgentDir(tenantUUID1))
		assert.NotContains(t, relevantDirs, cleaner.path.OsAgentDir(tenantUUID2))
	})
}

func createHostMonDk(t *testing.T, name, apiURL string) dynakube.DynaKube {
	t.Helper()

	dk := createBaseDk(t, name, apiURL)
	dk.Spec.OneAgent.HostMonitoring = &oneagent.HostInjectSpec{}

	return dk
}

func createCloudNativeDk(t *testing.T, name, apiURL string) dynakube.DynaKube {
	t.Helper()

	dk := createBaseDk(t, name, apiURL)
	dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}

	return dk
}
