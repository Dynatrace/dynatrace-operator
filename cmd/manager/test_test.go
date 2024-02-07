package manager

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestManager(t *testing.T) {
	mgr := TestManager{}

	assert.NotNil(t, mgr.GetClient())
	assert.NotNil(t, mgr.GetControllerOptions())
	assert.Equal(t, scheme.Scheme, mgr.GetScheme())
	assert.NotNil(t, mgr.GetLogger())
	require.NoError(t, mgr.Add(nil))
	require.NoError(t, mgr.Start(context.TODO()))
}
