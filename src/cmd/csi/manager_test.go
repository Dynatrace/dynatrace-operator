package csi

import (
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCsiDriverManagerProvider(t *testing.T) {
	t.Run("is instantiable", func(t *testing.T) {
		csiManagerProvider := newCsiDriverManagerProvider(defaultProbeAddress)
		assert.NotNil(t, csiManagerProvider)

		csiManagerProviderImpl := csiManagerProvider.(csiDriverManagerProvider)
		assert.Equal(t, defaultProbeAddress, csiManagerProviderImpl.probeAddress)
	})
	t.Run("creates options", func(t *testing.T) {
		csiManagerProvider := csiDriverManagerProvider{}

		options := csiManagerProvider.createOptions("namespace")

		assert.NotNil(t, options)
		assert.Equal(t, "namespace", options.Namespace)
		assert.Equal(t, scheme.Scheme, options.Scheme)
		assert.Equal(t, metricsBindAddress, options.MetricsBindAddress)
		assert.Equal(t, port, options.Port)
		assert.Equal(t, "", options.HealthProbeBindAddress)
		assert.Equal(t, livenessEndpointName, options.LivenessEndpointName)
	})

}
