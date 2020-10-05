package activegate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePod(t *testing.T) {
	r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NotNil(t, r)
	assert.NoError(t, err)

	pods, err := r.findPods(instance)
	assert.NoError(t, err)
	assert.NotEmpty(t, pods)

	lenBefore := len(pods)
	result, err := r.createPod(r.newPodForCR(instance, nil))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	pods, err = r.findPods(instance)
	assert.NoError(t, err)
	assert.NotEmpty(t, pods)
	assert.Equal(t, lenBefore+1, len(pods))
}
