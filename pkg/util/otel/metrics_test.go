package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestSetupMetrics(t *testing.T) {
	t.Run("do not use OTel", func(t *testing.T) {
		resource, err := newResource("testOtel")
		require.NoError(t, err)

		meterProvider, shutdownFn, err := setupMetrics(context.Background(), resource, "", "")
		require.NoError(t, err)

		assert.Nil(t, shutdownFn(context.Background()))
		assert.IsType(t, noop.NewMeterProvider(), meterProvider)
		assert.Equal(t, otel.GetMeterProvider(), meterProvider)
	})
}
