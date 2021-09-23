package mapper

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
)

func TestMatchForNamespaceNothingEverything(t *testing.T) {
	matchLabels := map[string]string{
		"type":   "app",
		"inject": "true",
	}
	dynakubes := []*dynatracev1alpha1.DynaKube{
		createTestDynakubeWithCodeModules("codeModules-1", nil, nil),
		createTestDynakubeWithCodeModules("codeModules-2", matchLabels, nil),
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
	dk := createTestDynakubeWithMultipleFeatures("codeModules-1", labels, nil)
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
		dk2 := createTestDynakubeWithCodeModules("codeModules-2", labels, nil)
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
	t.Run("Allow multiple dynakubes with different features", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		differentDk1 := createTestDynakubeWithDataIngest("dk1", labels, nil)
		differentDk2 := createTestDynakubeWithCodeModules("dk2", labels, nil)
		namespace := createNamespace("test-namespace", labels)
		clt := fake.NewClient(differentDk1, differentDk2)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()

		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, 2, len(nm.targetNs.Labels))
	})
}
