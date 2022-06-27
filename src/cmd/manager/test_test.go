package manager

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
)

func TestTestManager(t *testing.T) {
	mgr := TestManager{}

	assert.NotNil(t, mgr.GetClient())
	assert.NotNil(t, mgr.GetAPIReader())
	assert.NotNil(t, mgr.GetControllerOptions())
	assert.Equal(t, scheme.Scheme, mgr.GetScheme())
	assert.NotNil(t, mgr.GetLogger())
	assert.NoError(t, mgr.Add(nil))
	assert.NoError(t, mgr.Start(context.TODO()))
}
