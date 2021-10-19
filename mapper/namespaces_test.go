package mapper

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
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
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.updateNamespace()
		assert.NoError(t, err)
		assert.True(t, updated)
	})
}

func TestMapFromNamespace(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := createTestDynakubeWithMultipleFeatures("appMonitoring-1", labels, nil)
	namespace := createNamespace("test-namespace", labels)

	t.Run("Add to namespace", func(t *testing.T) {
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()

		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, 2, len(nm.targetNs.Labels))
	})

	t.Run("Error, 2 dynakube point to same namespace", func(t *testing.T) {
		dk2 := createTestDynakubeWithAppInject("appMonitoring-2", labels, nil)
		clt := fake.NewClient(dk, dk2)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()

		assert.Error(t, err)
		assert.False(t, updated)
	})

	t.Run("Remove stale namespace entry", func(t *testing.T) {
		labels := map[string]string{
			InstanceLabel: dk.Name,
		}
		namespace := createNamespace("test-namespace", labels)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()

		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, 0, len(nm.targetNs.Labels))
	})

	t.Run("Ignore kube namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil, nil)
		namespace := createNamespace("kube-something", nil)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()

		assert.NoError(t, err)
		assert.False(t, updated)
		assert.Equal(t, 0, len(nm.targetNs.Labels))
	})

	t.Run("Ignore openshift namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil, nil)
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()

		assert.NoError(t, err)
		assert.False(t, updated)
		assert.Equal(t, 0, len(nm.targetNs.Labels))
	})

	t.Run("Feature flag for monitoring system namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil, nil)
		dk.Annotations = map[string]string{
			"alpha.operator.dynatrace.com/feature-ignored-namespaces": "[]",
		}
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()

		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, 1, len(nm.targetNs.Labels))
	})
}
