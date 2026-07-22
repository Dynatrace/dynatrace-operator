// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package attributes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// newTestPodAttributes creates a PodAttributes with all maps initialized so tests can set
// individual fields without triggering nil-map panics.
func newTestPodAttributes() *Pod {
	return &Pod{
		rules:                make(map[string]string),
		namespaceAnnotations: make(map[string]string),
		podAnnotations:       make(map[string]string),
		dynakube:             make(map[string]string),
		custom:               make(map[string]string),
		workloadInfo:         make(map[string]string),
		clusterInfo:          make(map[string]string),
		podInfo:              make(map[string]string),
		deprecated:           make(map[string]string),
		podEnvVars:           []corev1.EnvVar{},
	}
}

// toResultMap converts the slice of "key=value" strings produced by Convert into a map.
func toResultMap(pairs []string) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, _ := strings.Cut(p, "=")
		m[k] = v
	}

	return m
}

// simpleConvertFunc formats each attribute as "key=value".
func simpleConvertFunc(k, v string) string { return k + "=" + v }

func TestSetCustomAttributes(t *testing.T) {
	t.Run("bulk copies all entries into custom", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.SetCustomAttributes(map[string]string{"a": "1", "b": "2"})
		assert.Equal(t, "1", attrs.custom["a"])
		assert.Equal(t, "2", attrs.custom["b"])
	})
}

func TestGetPodEnvVars(t *testing.T) {
	t.Run("returns the internal podEnvVars slice", func(t *testing.T) {
		attrs := newTestPodAttributes()
		env := corev1.EnvVar{Name: "FOO", Value: "bar"}
		attrs.podEnvVars = append(attrs.podEnvVars, env)
		result := attrs.GetPodEnvVars()
		require.Len(t, result, 1)
		assert.Equal(t, env, result[0])
	})

	t.Run("returns empty slice when no env vars set", func(t *testing.T) {
		attrs := newTestPodAttributes()
		assert.Empty(t, attrs.GetPodEnvVars())
	})
}

func TestConvert_Method(t *testing.T) {
	t.Run("combines and converts attributes to key=value strings", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.workloadInfo["k8s.workload.kind"] = "deployment"
		attrs.custom["my.attr"] = "custom-val"

		result := attrs.Convert(simpleConvertFunc)

		m := toResultMap(result)
		assert.Equal(t, "deployment", m["k8s.workload.kind"])
		assert.Equal(t, "custom-val", m["my.attr"])
	})

	t.Run("passes ContainerAttributes into the result", func(t *testing.T) {
		attrs := newTestPodAttributes()
		container := Container{ContainerName: "my-container"}

		result := attrs.Convert(simpleConvertFunc, container)

		m := toResultMap(result)
		assert.Equal(t, "my-container", m[K8sContainerNameAttr])
	})
}

func TestConvert(t *testing.T) {
	t.Run("applies convertFunc to each entry and returns all results", func(t *testing.T) {
		attrs := map[string]string{
			"key1": "val1",
			"key2": "val2",
		}
		result := convert(attrs, func(k, v string) string { return k + "=" + v })
		assert.ElementsMatch(t, []string{"key1=val1", "key2=val2"}, result)
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		result := convert(map[string]string{}, func(k, v string) string { return k + "=" + v })
		assert.Empty(t, result)
	})

	t.Run("convertFunc receives both key and value", func(t *testing.T) {
		attrs := map[string]string{"mykey": "myval"}
		var gotKey, gotVal string
		convert(attrs, func(k, v string) string {
			gotKey, gotVal = k, v

			return ""
		})
		assert.Equal(t, "mykey", gotKey)
		assert.Equal(t, "myval", gotVal)
	})

	// Regression: convert() must not include empty strings returned by the convertFunc.
	// If it did, strings.Join(result, ",") would produce spurious commas in
	// OTEL_RESOURCE_ATTRIBUTES (e.g. "k1=v1,,k2=v2") for attributes with empty values
	// such as dt.entity.kubernetes_cluster when KubernetesClusterMEID is not yet set.
	t.Run("empty return values from convertFunc are excluded", func(t *testing.T) {
		attrs := map[string]string{
			"key-with-value": "val",
			"key-empty":      "",
		}
		result := convert(attrs, func(k, v string) string {
			if v == "" {
				return ""
			}

			return k + "=" + v
		})
		assert.Equal(t, []string{"key-with-value=val"}, result)
	})
}
