package activegate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeletePods(t *testing.T) {
	t.Run("UpdatePods", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NoError(t, err)

		pods, err := r.findPods(instance)
		assert.NotEmpty(t, pods)
		assert.NoError(t, err)

		err = r.deletePods(log.WithName("DeletePods"), pods)
		assert.NoError(t, err)

		pods, err = r.findPods(instance)
		assert.NoError(t, err)
		assert.Empty(t, pods)
	})
}
