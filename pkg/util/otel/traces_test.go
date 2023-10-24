package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/context"
)

func TestSetupTraces(t *testing.T) {
	t.Run("do not use OTel", func(t *testing.T) {
		resource, err := newResource("testOtel")
		require.NoError(t, err)

		traceProvider, shutdownFn, err := setupTraces(context.Background(), resource, "", "")
		require.NoError(t, err)

		assert.Nil(t, shutdownFn(context.Background()))
		assert.IsType(t, trace.NewNoopTracerProvider(), traceProvider)
		assert.Equal(t, otel.GetTracerProvider(), traceProvider)
	})
}

func TestStartSpan(t *testing.T) {
	t.Run("nil tracer", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), nil, "testSpan")

		assert.IsType(t, noopSpan{}, span)
		assert.Equal(t, context.Background(), ctx)
	})
	t.Run("empty title", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), trace.NewNoopTracerProvider().Tracer("testTracer"), "")

		assert.IsType(t, noopSpan{}, span)
		assert.Equal(t, context.Background(), ctx)
	})
}
