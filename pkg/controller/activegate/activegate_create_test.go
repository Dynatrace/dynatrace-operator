package activegate

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreatePod(t *testing.T) {
	r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NotNil(t, r)
	assert.NoError(t, err)

	pods, err := r.findPods(instance)
	assert.Nil(t, err)
	assert.NotEmpty(t, pods)

	lenBefore := len(pods)
	result, err := r.createPod(r.newPodForCR(instance, nil))
	assert.Nil(t, err)
	assert.NotNil(t, result)

	pods, err = r.findPods(instance)
	assert.Nil(t, err)
	assert.NotEmpty(t, pods)
	assert.Equal(t, lenBefore+1, len(pods))
}
