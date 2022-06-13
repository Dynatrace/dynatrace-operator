package processmoduleconfig

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
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

func TestUpdateProcessModuleConfig(t *testing.T) {
	memFs := afero.NewMemMapFs()
	prepTestConfFs(memFs)
	expectedUsed := `
[general]
key value

[test]
test test3
`

	err := UpdateProcessModuleConfigInPlace(memFs, "", &testProcessModuleConfig)
	require.NoError(t, err)
	assertTestConf(t, memFs, ruxitAgentProcPath, expectedUsed)
	assertTestConf(t, memFs, sourceRuxitAgentProcPath, testRuxitConf)
}

func TestCheckProcessModuleConfigCopy(t *testing.T) {
	memFs := afero.NewMemMapFs()
	prepTestConfFs(memFs)
	sourcePath := sourceRuxitAgentProcPath
	destPath := ruxitAgentProcPath

	err := checkProcessModuleConfigCopy(memFs, sourcePath, destPath)
	require.NoError(t, err)
	assertTestConf(t, memFs, sourcePath, testRuxitConf)
}

func prepTestConfFs(fs afero.Fs) {
	_ = fs.MkdirAll(filepath.Base(sourceRuxitAgentProcPath), 0755)
	usedConf, _ := fs.OpenFile(ruxitAgentProcPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	_, _ = usedConf.WriteString(testRuxitConf)
}

func assertTestConf(t *testing.T, fs afero.Fs, path, expected string) {
	file, err := fs.Open(path)
	require.Nil(t, err)
	content, err := ioutil.ReadAll(file)
	require.Nil(t, err)
	assert.Equal(t, expected, string(content))
}
