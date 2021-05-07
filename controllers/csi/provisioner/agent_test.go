package csiprovisioner

import (
	"github.com/spf13/afero"
	assert "github.com/stretchr/testify/assert"
	"testing"
)

func TestOneAgentProvisioner_InstallAgent(t *testing.T) {
	fs := afero.NewMemMapFs()
	installAgentCfg := &installAgentConfig{
		fs: fs,
	}

	err := installAgent(installAgentCfg)
	assert.Error(t, err)
}
