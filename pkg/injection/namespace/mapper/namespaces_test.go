package mapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMatchForNamespaceNothingEverything(t *testing.T) {
	matchLabels := map[string]string{
		"type":   "app",
		"inject": "true",
	}
	dks := []*dynakube.DynaKube{
		createDynakubeWithAppInject("appMonitoring-1", metav1.LabelSelector{}),
		createDynakubeWithAppInject("appMonitoring-2", convertToLabelSelector(matchLabels)),
	}

	t.Run(`Match to unlabeled namespace`, func(t *testing.T) {
		namespace := createNamespace("test-namespace", nil)
		clt := fake.NewClient(dks[0], dks[1])
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.updateNamespace(context.Background())
		require.NoError(t, err)
		assert.True(t, updated)
	})
}

func TestMapFromNamespace(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := createDynakubeWithAppInject("appMonitoring-1", convertToLabelSelector(labels))
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
		dk2 := createDynakubeWithAppInject("appMonitoring-2", convertToLabelSelector(labels))
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
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("kube-something", nil)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.MapFromNamespace(context.Background())

		require.NoError(t, err)
		assert.False(t, updated)
		assert.Empty(t, nm.targetNs.Labels)
	})

	t.Run("Ignore openshift namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(clt, clt, "dynatrace", namespace)

		updated, err := nm.MapFromNamespace(context.Background())

		require.NoError(t, err)
		assert.False(t, updated)
		assert.Empty(t, nm.targetNs.Labels)
	})

	t.Run("ComponentFeature flag for monitoring system namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		dk.Annotations = map[string]string{
			exp.InjectionIgnoredNamespacesKey: "[]",
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
