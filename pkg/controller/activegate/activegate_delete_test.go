package activegate

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/controller/builder"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeletePods(t *testing.T) {
	t.Run("UpdatePods", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NoError(t, err)

		dummy := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name,
				Namespace: _const.DynatraceNamespace,
				Labels:    builder.BuildLabelsForQuery(instance.Name),
			},
		}
		r.client.Create(context.TODO(), &dummy)

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
