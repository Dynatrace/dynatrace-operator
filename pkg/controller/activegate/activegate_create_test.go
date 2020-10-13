package activegate

import (
	"testing"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/stretchr/testify/assert"
)

func TestCreateStatefulSet(t *testing.T) {
	r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NotNil(t, r)
	assert.NoError(t, err)

	result, err := r.newStatefulSetForCR(instance, &dtclient.TenantInfo{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
