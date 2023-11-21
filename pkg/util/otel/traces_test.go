package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/context"
)

func TestSetupTraces(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		resource, err := newResource("testOtel")
		require.NoError(t, err)

		tracerProvider, shutdownFn, err := setupTracesWithOtlp(context.Background(), resource, "abc12345.dynatrace.com", "dtabc.abc.abc")
		require.NoError(t, err)

		assert.NotNil(t, shutdownFn)
		assert.NotNil(t, tracerProvider)
	})
	t.Run("do not use OpenTelementry", func(t *testing.T) {
		resource, err := newResource("testOtel")
		require.NoError(t, err)

		tracerProvider, shutdownFn, err := setupTracesWithOtlp(context.Background(), resource, "", "")
		require.Error(t, err)

		assert.Nil(t, shutdownFn)
		assert.Nil(t, tracerProvider)
	})
}

func TestStartSpan(t *testing.T) {
	t.Run("nil tracer", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), trace.Tracer(nil), "testSpan")

		assert.IsType(t, noopSpan{}, span)
		assert.Equal(t, context.Background(), ctx)
	})
	t.Run("empty title", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), trace.NewNoopTracerProvider().Tracer("testTracer"), "")

		assert.IsType(t, noopSpan{}, span)
		assert.Equal(t, context.Background(), ctx)
	})
}
