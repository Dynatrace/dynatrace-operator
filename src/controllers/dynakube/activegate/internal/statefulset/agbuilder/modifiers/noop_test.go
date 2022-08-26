package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder"
	"github.com/stretchr/testify/assert"
)

func TestNoopModifier(t *testing.T) {
	t.Run("Run noop modifiers", func(t *testing.T) {
		const value = "bar"

		b := agbuilder.Builder{}
		b.AddModifier(NoopModifier{Msg: "foo"})

		// override annotation:
		b.AddModifier(NoopModifier{Msg: value})

		actual := b.Build()

		assert.Equal(t, 1, len(actual.Annotations))
		assert.True(t, actual.Annotations["noop"] == value)
	})
}
