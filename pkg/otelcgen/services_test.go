// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

func TestNewConfigWithServices(t *testing.T) {
	cfg, err := NewConfig(
		"",
		RegisteredProtocols,
		WithServices(),
	)
	require.NoError(t, err)
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "services_only.yaml"))
	require.NoError(t, err)
	assert.YAMLEq(t, string(expectedOutput), string(c))
}

func TestNewConfigWithServicesZipkinOnly(t *testing.T) {
	cfg, err := NewConfig(
		"",
		Protocols{ZipkinProtocol},
		WithServices(),
	)
	require.NoError(t, err)
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "services_zipkin_only.yaml"))
	require.NoError(t, err)
	assert.YAMLEq(t, string(expectedOutput), string(c))
}

func TestNewConfigWithServicesStatsdOnly(t *testing.T) {
	cfg, err := NewConfig(
		"",
		Protocols{StatsdProtocol},
		WithServices(),
	)
	require.NoError(t, err)
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "services_statsd_only.yaml"))
	require.NoError(t, err)
	assert.YAMLEq(t, string(expectedOutput), string(c))
}

func TestBuildPipelineProcessors_WithResourceAttributes(t *testing.T) {
	cfg := &Config{resourceAttributes: map[string]string{"team": "shopping-cart"}}

	processors := cfg.buildPipelineProcessors()

	require.Equal(t, []component.ID{
		memoryLimiter, transformPodIP, k8sattributesAnnotations, staticResourceAttrs, k8sattributesFacts, transform,
	}, processors)
}

func TestBuildPipelineProcessors_WithoutResourceAttributes(t *testing.T) {
	cfg := &Config{}

	processors := cfg.buildPipelineProcessors()

	require.Equal(t, []component.ID{
		memoryLimiter, transformPodIP, k8sattributesAnnotations, k8sattributesFacts, transform,
	}, processors)
}
