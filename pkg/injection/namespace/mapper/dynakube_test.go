package mapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestMapFromDynakube(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(labels))
	namespace := createNamespace("test-namespace", labels)

	t.Run("Add to namespace", func(t *testing.T) {
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Len(t, ns.Labels, 2)
		assert.Len(t, ns.Annotations, 1)
	})
	t.Run("Overwrite stale entry in labels", func(t *testing.T) {
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: "old-dk",
			"test":                           "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Len(t, ns.Labels, 2)
		assert.Len(t, ns.Annotations, 1)
	})
	t.Run("Remove stale dynakube entry for no longer matching ns", func(t *testing.T) {
		movedDk := createDynakubeWithAppInject("moved-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: movedDk.Name,
		}
		namespace := createNamespace("test-namespace", nsLabels)
		clt := fake.NewClient(movedDk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", movedDk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
		assert.Len(t, ns.Annotations, 1)
	})
	t.Run("Throw error in case of conflicting Dynakubes", func(t *testing.T) {
		conflictingDk := createDynakubeWithAppInject("conflicting-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: dk.Name,
			"test":                           "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)
		clt := fake.NewClient(dk, conflictingDk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", conflictingDk)

		err := dm.MapFromDynakube()

		require.Error(t, err)
	})
	t.Run("Ignore kube namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("kube-something", nil)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
		assert.Empty(t, ns.Annotations)
	})

	t.Run("Ignore openshift namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
		assert.Empty(t, ns.Annotations)
	})
	t.Run("ComponentFeature flag for monitoring system namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		dk.Annotations = map[string]string{
			exp.IgnoredNamespacesAnnotation: "[]",
		}
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Len(t, ns.Labels, 1)
		assert.Len(t, ns.Annotations, 1)
	})
}

func TestUnmapFromDynaKube(t *testing.T) {
	dk := createDynakubeWithAppInject("dk", metav1.LabelSelector{})
	labels := map[string]string{
		dtwebhook.InjectionInstanceLabel: dk.Name,
	}
	namespace := createNamespace("ns1", labels)
	namespace2 := createNamespace("ns2", labels)

	t.Run("Remove from no ns => no error", func(t *testing.T) {
		clt := fake.NewClient()

		namespaces, err := GetNamespacesForDynakube(context.Background(), clt, dk.Name)
		require.NoError(t, err)

		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk)
		err = dm.UnmapFromDynaKube(namespaces)
		require.NoError(t, err)
	})
	t.Run("Remove from everywhere, multiple entries", func(t *testing.T) {
		clt := fake.NewClient(namespace, namespace2)

		namespaces, err := GetNamespacesForDynakube(context.Background(), clt, dk.Name)
		require.NoError(t, err)

		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk)
		err = dm.UnmapFromDynaKube(namespaces)
		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
		assert.Len(t, ns.Annotations, 1)
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace2.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
		assert.Len(t, ns.Annotations, 1)
	})
}
