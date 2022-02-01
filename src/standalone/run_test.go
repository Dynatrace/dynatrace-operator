package standalone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {

	runner := creatTestRunner(t)

	assert.NotNil(t, runner.fs)
	assert.NotNil(t, runner.env)
	assert.NotNil(t, runner.client)
	assert.NotNil(t, runner.config)
	assert.NotNil(t, runner.installer)
	assert.Empty(t, runner.hostTenant)
}

func creatTestRunner(t *testing.T) *Runner {
	fs := prepTestFs(t)
	resetEnv := prepTestEnv(t)

	runner, err := NewRunner(fs)
	resetEnv()

	require.NoError(t, err)
	require.NotNil(t, runner)

	return runner
}
