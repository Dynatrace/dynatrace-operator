package dtotel

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
		ctx, span := StartSpan(context.Background(), trace.Tracer(nil))

		assert.Equal(t, context.Background(), ctx)

		assert.IsType(t, noopSpan{}, span)
		noopSpan, ok := span.(noopSpan)
		require.True(t, ok)
		assert.Contains(t, noopSpan.spanTitle, "dtotel.TestStartSpan.func")
		assert.NotContains(t, noopSpan.spanTitle, "dtotel.StartSpan")
	})
}

func TestIsEnabled(t *testing.T) {
	t.Run("span context is empty", func(t *testing.T) {
		spanContext := trace.SpanContext{}

		assert.False(t, IsEnabled(spanContext))
	})
	t.Run("span context is not empty", func(t *testing.T) {
		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{1, 2, 3},
			SpanID:  trace.SpanID{4, 5, 6},
		})

		assert.True(t, IsEnabled(spanContext))
	})
}

func TestGetCaller(t *testing.T) {
	// Call getCaller with stack depth of 1
	caller := getCaller(1)

	t.Logf("calloer=%s", caller)

	// Assert caller contains function name
	assert.Contains(t, caller, "TestGetCaller")

	// Assert caller contains file name
	assert.Contains(t, caller, "traces_test.go")

	// Assert caller contains line number
	assert.Regexp(t, `\(\w+_test.go:\d+\)`, caller)
}
