package activegate

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeletePods(t *testing.T) {
	t.Run("UpdatePods", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NoError(t, err)

		pods, err := r.findPods(instance)
		assert.Nil(t, err)
		assert.NotEmpty(t, pods)

		err = r.deletePods(log.WithName("DeletePods"), pods)
		assert.Nil(t, err)

		pods, err = r.findPods(instance)
		assert.Nil(t, err)
		assert.Empty(t, pods)
	})
}
