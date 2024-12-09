package processmoduleconfig

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRuxitConf = `
[general]
key value
`
)

var testProcessModuleConfig = dtclient.ProcessModuleConfig{
	Revision: 3,
	Properties: []dtclient.ProcessModuleProperty{
		{
			Section: "test",
			Key:     "test",
			Value:   "test3",
		},
	},
}

func TestUpdateProcessModuleConfigInPlace(t *testing.T) {
	t.Run("no processModuleConfig", func(t *testing.T) {
		memFs := afero.NewMemMapFs()

		err := UpdateInPlace(memFs, "", nil)
		require.NoError(t, err)
	})
	t.Run("update file", func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		prepTestConfFs(memFs)

		expectedUsed := `
[general]
key value

[test]
test test3
`

		err := UpdateInPlace(memFs, "", &testProcessModuleConfig)
		require.NoError(t, err)
		assertTestConf(t, memFs, RuxitAgentProcPath, expectedUsed)
		assertTestConf(t, memFs, sourceRuxitAgentProcPath, testRuxitConf)
	})
}

func TestCreateAgentConfigDir(t *testing.T) {
	t.Run("no processModuleConfig", func(t *testing.T) {
		memFs := afero.NewMemMapFs()

		err := UpdateFromDir(memFs, "", "", nil)
		require.NoError(t, err)
	})

	t.Run("create config dir with file", func(t *testing.T) {
		targetDir := "test"
		sourceDir := ""
		memFs := afero.NewMemMapFs()
		prepTestConfFs(memFs)

		expectedUsed := `
[general]
key value

[test]
test test3
`

		err := UpdateFromDir(memFs, targetDir, sourceDir, &testProcessModuleConfig)
		require.NoError(t, err)
		assertTestConf(t, memFs, filepath.Join(targetDir, RuxitAgentProcPath), expectedUsed)
		assertTestConf(t, memFs, filepath.Join(sourceDir, RuxitAgentProcPath), testRuxitConf)
	})
}

func TestCheckProcessModuleConfigCopy(t *testing.T) {
	memFs := afero.NewMemMapFs()
	prepTestConfFs(memFs)

	sourcePath := sourceRuxitAgentProcPath
	destPath := RuxitAgentProcPath

	err := checkProcessModuleConfigCopy(memFs, sourcePath, destPath)
	require.NoError(t, err)
	assertTestConf(t, memFs, sourcePath, testRuxitConf)
}

func prepTestConfFs(fs afero.Fs) {
	_ = fs.MkdirAll(filepath.Base(sourceRuxitAgentProcPath), 0755)
	_ = fs.MkdirAll(filepath.Base(RuxitAgentProcPath), 0755)
	usedConf, _ := fs.OpenFile(RuxitAgentProcPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	_, _ = usedConf.WriteString(testRuxitConf)
}

func assertTestConf(t *testing.T, fs afero.Fs, path, expected string) {
	file, err := fs.Open(path)
	require.NoError(t, err)
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, expected, string(content))
}
