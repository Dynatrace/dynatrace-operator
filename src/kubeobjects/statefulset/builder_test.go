package statefulset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestStatefulsetBuilder(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		b := Builder{}
		actual := b.Build()
		expected := appsv1.StatefulSet{}
		assert.Equal(t, expected, actual)
	})
}
