package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveDeprecatedMounts(t *testing.T) {
	t.Run("empty fsState -> no panic", func(t *testing.T) {
		cleaner := createCleaner(t)

		cleaner.removeDeprecatedMounts(fsState{})
	})

	t.Run("remove unmounted deprecated dirs (empty mapped subdir)", func(t *testing.T) {
		cleaner := createCleaner(t)

		deprecateDir := []string{
			"t0",
			"t1",
			"t2",
		}

		for i, folder := range deprecateDir {
			cleaner.createDeprecatedDirs(t, folder, i, i*2)

			expectedDir := cleaner.path.AgentRunDir(folder)
			require.DirExists(t, expectedDir)

			subdirs, err := os.ReadDir(expectedDir)
			require.NoError(t, err)
			require.Len(t, subdirs, i+i*2)
		}

		cleaner.removeDeprecatedMounts(fsState{
			deprecatedDks: deprecateDir,
		})

		for i, folder := range deprecateDir {
			expectedDir := cleaner.path.AgentRunDir(folder)

			if i == 0 {
				assert.NoDirExists(t, expectedDir)
			} else {
				require.DirExists(t, expectedDir)

				subdirs, err := os.ReadDir(expectedDir)
				require.NoError(t, err)
				require.Len(t, subdirs, i)
			}
		}
	})
}

func (c *Cleaner) createDeprecatedDirs(t *testing.T, name string, subDirAmount, emptySubDirAmount int) {
	t.Helper()

	runDir := c.path.AgentRunDir(name)
	err := os.MkdirAll(runDir, os.ModePerm)
	require.NoError(t, err)

	for i := range subDirAmount {
		mappedDir := c.path.OverlayMappedDir(name, fmt.Sprintf("volume-%d", i))
		err := os.MkdirAll(mappedDir, os.ModePerm)
		require.NoError(t, err)
		file, err := os.Create(filepath.Join(mappedDir, "something"))
		require.NoError(t, err)
		_, err = file.WriteString("something")
		require.NoError(t, err)
	}

	for i := range emptySubDirAmount {
		mappedDir := c.path.OverlayMappedDir(name, fmt.Sprintf("volume-%d", i+subDirAmount))
		err := os.MkdirAll(mappedDir, os.ModePerm)
		require.NoError(t, err)
	}
}
