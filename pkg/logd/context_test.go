package logd_test

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func TestFromContext(t *testing.T) {
	t.Run("returns fallback logger when ctx is nil", func(t *testing.T) {
		log := logd.FromContext(nil) //nolint:staticcheck

		assert.NotNil(t, log.GetSink())
	})

	t.Run("returns fallback logger when context has no logger", func(t *testing.T) {
		log := logd.FromContext(t.Context())

		assert.NotNil(t, log.GetSink())
	})

	t.Run("returns injected logger when context carries one", func(t *testing.T) {
		injected := logd.Get().WithName("test")
		ctx := logr.NewContext(t.Context(), injected.Logger)

		log := logd.FromContext(ctx)

		assert.Equal(t, injected.Logger, log.Logger)
	})
}

func TestNewFromContext(t *testing.T) {
	t.Run("falls back to logd.Get() when context has no logger", func(t *testing.T) {
		ctx, log := logd.NewFromContext(t.Context(), "myname")

		assert.NotNil(t, log.GetSink())
		// The returned context must carry the derived logger.
		fromCtx := logd.FromContext(ctx)
		assert.NotNil(t, fromCtx.GetSink())
	})

	t.Run("derives from existing logger in context", func(t *testing.T) {
		base := logd.Get().WithName("base")
		ctx := logr.NewContext(t.Context(), base.Logger)

		newCtx, log := logd.NewFromContext(ctx, "child", "key", "value")

		assert.NotNil(t, log.GetSink())
		// The new context must carry the derived logger.
		fromCtx := logd.FromContext(newCtx)
		assert.NotNil(t, fromCtx.GetSink())
	})

	t.Run("stores derived logger back into context", func(t *testing.T) {
		ctx, derived := logd.NewFromContext(t.Context(), "stored")

		retrieved := logd.FromContext(ctx)

		assert.Equal(t, derived.Logger, retrieved.Logger)
	})
}

func TestIntoContext(t *testing.T) {
	t.Run("stores logger into context and retrieves it", func(t *testing.T) {
		log := logd.Get().WithName("into-test")
		ctx := logd.IntoContext(t.Context(), log)

		retrieved := logd.FromContext(ctx)

		assert.Equal(t, log.Logger, retrieved.Logger)
	})
}
