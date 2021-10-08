package mapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestMapFromDynakube(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := createTestDynakubeWithMultipleFeatures("dk-test", labels, nil)
	namespace := createNamespace("test-namespace", labels)

	t.Run("Add to namespace", func(t *testing.T) {
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk, logger.NewDTLogger())

		err := dm.MapFromDynakube()

		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(ns.Labels))
		assert.Equal(t, 1, len(ns.Annotations))
	})
	t.Run("Overwrite stale entry in labels", func(t *testing.T) {
		nsLabels := map[string]string{
			InstanceLabel: "old-dk",
			"test":        "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk, logger.NewDTLogger())

		err := dm.MapFromDynakube()

		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(ns.Labels))
		assert.Equal(t, 1, len(ns.Annotations))
	})
	t.Run("Remove stale dynakube entry for no longer matching ns", func(t *testing.T) {
		movedDk := createTestDynakubeWithAppInject("moved-dk", labels, nil)
		nsLabels := map[string]string{
			InstanceLabel: movedDk.Name,
		}
		namespace := createNamespace("test-namespace", nsLabels)
		clt := fake.NewClient(movedDk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", movedDk, logger.NewDTLogger())

		err := dm.MapFromDynakube()

		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(ns.Labels))
		assert.Equal(t, 1, len(ns.Annotations))
	})
	t.Run("Throw error in case of conflicting Dynakubes", func(t *testing.T) {
		conflictingDk := createTestDynakubeWithMultipleFeatures("conflicting-dk", labels, nil)
		nsLabels := map[string]string{
			InstanceLabel: dk.Name,
			"test":        "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)
		clt := fake.NewClient(dk, conflictingDk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", conflictingDk, logger.NewDTLogger())

		err := dm.MapFromDynakube()

		assert.Error(t, err)
	})
	t.Run("Ignore kube namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil, nil)
		namespace := createNamespace("kube-something", nil)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk, logger.NewDTLogger())

		err := dm.MapFromDynakube()

		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(ns.Labels))
		assert.Equal(t, 0, len(ns.Annotations))
	})

	t.Run("Ignore openshift namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil, nil)
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk, logger.NewDTLogger())

		err := dm.MapFromDynakube()

		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(ns.Labels))
		assert.Equal(t, 0, len(ns.Annotations))
	})
}

func TestUnmapFromDynaKube(t *testing.T) {
	dk := createTestDynakubeWithAppInject("dk", nil, nil)
	labels := map[string]string{
		InstanceLabel: dk.Name,
	}
	namespace := createNamespace("ns1", labels)
	namespace2 := createNamespace("ns2", labels)

	t.Run("Remove from no ns => no error", func(t *testing.T) {
		clt := fake.NewClient()
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk, logger.NewDTLogger())
		err := dm.UnmapFromDynaKube()
		assert.NoError(t, err)
	})
	t.Run("Remove from everywhere, multiple entries", func(t *testing.T) {
		clt := fake.NewClient(namespace, namespace2)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk, logger.NewDTLogger())
		err := dm.UnmapFromDynaKube()
		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(ns.Labels))
		assert.Equal(t, 1, len(ns.Annotations))
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace2.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(ns.Labels))
		assert.Equal(t, 1, len(ns.Annotations))
	})
}
