package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
			exists, _ := cleaner.fs.Exists(expectedDir)
			require.True(t, exists)

			subdirs, err := cleaner.fs.ReadDir(expectedDir)
			require.NoError(t, err)
			require.Len(t, subdirs, i+i*2)
		}

		cleaner.removeDeprecatedMounts(fsState{
			deprecatedDks: deprecateDir,
		})

		for i, folder := range deprecateDir {
			expectedDir := cleaner.path.AgentRunDir(folder)

			exists, _ := cleaner.fs.Exists(expectedDir)
			if i == 0 {
				require.False(t, exists)
			} else {
				require.True(t, exists)

				subdirs, err := cleaner.fs.ReadDir(expectedDir)
				require.NoError(t, err)
				require.Len(t, subdirs, i)
			}
		}
	})
}

func (c *Cleaner) createDeprecatedDirs(t *testing.T, name string, subDirAmount, emptySubDirAmount int) {
	t.Helper()

	runDir := c.path.AgentRunDir(name)
	err := c.fs.MkdirAll(runDir, os.ModePerm)
	require.NoError(t, err)

	for i := range subDirAmount {
		mappedDir := c.path.OverlayMappedDir(name, fmt.Sprintf("volume-%d", i))
		err := c.fs.MkdirAll(mappedDir, os.ModePerm)
		require.NoError(t, err)
		file, err := c.fs.Create(filepath.Join(mappedDir, "something"))
		require.NoError(t, err)
		_, err = file.WriteString("something")
		require.NoError(t, err)
	}

	for i := range emptySubDirAmount {
		mappedDir := c.path.OverlayMappedDir(name, fmt.Sprintf("volume-%d", i+subDirAmount))
		err := c.fs.MkdirAll(mappedDir, os.ModePerm)
		require.NoError(t, err)
	}
}
