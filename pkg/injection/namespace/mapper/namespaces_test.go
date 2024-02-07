package mapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchForNamespaceNothingEverything(t *testing.T) {
	matchLabels := map[string]string{
		"type":   "app",
		"inject": "true",
	}
	dynakubes := []*dynatracev1beta1.DynaKube{
		createTestDynakubeWithAppInject("appMonitoring-1", nil, nil),
		createTestDynakubeWithAppInject("appMonitoring-2", matchLabels, nil),
	}

	t.Run(`Match to unlabeled namespace`, func(t *testing.T) {
		namespace := createNamespace("test-namespace", nil)
		clt := fake.NewClient(dynakubes[0], dynakubes[1])
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.updateNamespace(context.Background())
		require.NoError(t, err)
		assert.True(t, updated)
	})
}

func TestMapFromNamespace(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := createTestDynakubeWithMultipleFeatures("appMonitoring-1", labels)
	namespace := createNamespace("test-namespace", labels)

	t.Run("Add to namespace", func(t *testing.T) {
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.MapFromNamespace(context.Background())

		require.NoError(t, err)
		assert.True(t, updated)
		assert.Len(t, nm.targetNs.Labels, 2)
	})

	t.Run("Error, 2 dynakubes point to same namespace", func(t *testing.T) {
		dk2 := createTestDynakubeWithAppInject("appMonitoring-2", labels, nil)
		clt := fake.NewClient(dk, dk2)
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.MapFromNamespace(context.Background())

		require.Error(t, err)
		assert.False(t, updated)
	})

	t.Run("Remove stale namespace entry", func(t *testing.T) {
		labels := map[string]string{
			dtwebhook.InjectionInstanceLabel: dk.Name,
		}
		namespace := createNamespace("test-namespace", labels)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.MapFromNamespace(context.Background())

		require.NoError(t, err)
		assert.True(t, updated)
		assert.Empty(t, nm.targetNs.Labels)
	})

	t.Run("Ignore kube namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil)
		namespace := createNamespace("kube-something", nil)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.MapFromNamespace(context.Background())

		require.NoError(t, err)
		assert.False(t, updated)
		assert.Empty(t, nm.targetNs.Labels)
	})

	t.Run("Ignore openshift namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil)
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.MapFromNamespace(context.Background())

		require.NoError(t, err)
		assert.False(t, updated)
		assert.Empty(t, nm.targetNs.Labels)
	})

	t.Run("ComponentFeature flag for monitoring system namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil)
		dk.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureIgnoredNamespaces: "[]",
		}
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.MapFromNamespace(context.Background())

		require.NoError(t, err)
		assert.True(t, updated)
		assert.Len(t, nm.targetNs.Labels, 1)
	})
}
