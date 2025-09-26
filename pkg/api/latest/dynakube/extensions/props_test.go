package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseSpec_GetReplicas(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		d := DatabaseSpec{Replicas: nil}

		assert.Equal(t, int32(1), d.GetReplicas())
	})
	t.Run("non-nil value", func(t *testing.T) {
		replicas := int32(2)
		d := DatabaseSpec{Replicas: &replicas}

		assert.Equal(t, int32(2), d.GetReplicas())
	})
}
