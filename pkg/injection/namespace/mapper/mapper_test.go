package mapper

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createBaseDynakube(name string, appInjection bool, metadataEnrichment bool) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dynatrace"},
		Spec: dynakube.DynaKubeSpec{
			MetadataEnrichment: dynakube.MetadataEnrichment{
				Enabled: &metadataEnrichment,
			},
		},
	}

	if appInjection {
		dk.Spec.OneAgent = oneagent.Spec{
			ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{}}
	}

	return dk
}

func createDynakubeWithAppInject(name string, selector metav1.LabelSelector) *dynakube.DynaKube {
	dk := createBaseDynakube(name, true, false)
	dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = selector

	return dk
}

func createDynakubeWithMetadataEnrichment(name string, selector metav1.LabelSelector) *dynakube.DynaKube {
	dk := createBaseDynakube(name, false, true)
	dk.Spec.MetadataEnrichment.NamespaceSelector = selector

	return dk
}

func createDynakubeWithMetadataAndAppInjection(name string, selector metav1.LabelSelector) *dynakube.DynaKube {
	dk := createBaseDynakube(name, true, true)

	dk.Spec.MetadataEnrichment.NamespaceSelector = selector
	dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = selector

	return dk
}

func createDynakubeWithNodeImagePullAndNoCSI(name string, selector metav1.LabelSelector) *dynakube.DynaKube {
	dk := createBaseDynakube(name, true, false)

	dk.Annotations = make(map[string]string)

	dk.Annotations[exp.OANodeImagePullKey] = "true"

	dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = selector

	return dk
}

func createNamespace(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func convertToLabelSelector(labels map[string]string) metav1.LabelSelector {
	return metav1.LabelSelector{MatchLabels: labels}
}

func TestUpdateNamespace(t *testing.T) {
	t.Run("Add to namespace", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(labels))
		namespace := createNamespace("test-namespace", labels)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
	})
	t.Run("Add to namespace, when only metadata enrichment is enabled", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithMetadataEnrichment("dk-test", convertToLabelSelector(labels))
		namespace := createNamespace("test-namespace", labels)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
	})
	t.Run("Add to namespace, with metadata and appInjection enabled", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithMetadataAndAppInjection("appMonitoring", convertToLabelSelector(labels))

		namespace := createNamespace("test", labels)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Equal(t, dk.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
	t.Run("Overwrite stale entry in labels", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: "old-dk",
			"test":                           "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
		assert.Equal(t, dk.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
	t.Run("Remove stale dynakube entry for no longer matching ns", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		movedDk := createDynakubeWithAppInject("moved-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: movedDk.Name,
		}
		namespace := createNamespace("test-namespace", nsLabels)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*movedDk}})
		require.NoError(t, err)
		require.True(t, updated)
		assert.Empty(t, namespace.Labels)
	})
	t.Run("Throw error in case of conflicting Dynakubes", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(labels))
		conflictingDk := createDynakubeWithMetadataEnrichment("conflicting-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: dk.Name,
			"test":                           "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)

		_, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*conflictingDk, *dk}})

		require.Error(t, err)
	})
	t.Run("Ignore kube namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("kube-something", nil)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.False(t, updated)
		assert.Empty(t, namespace.Labels)
	})

	t.Run("Ignore openshift namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("openshift-something", nil)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.False(t, updated)
		assert.Empty(t, namespace.Labels)
	})
	t.Run("Double dynakube, 1. ignores openshift namespaces, 2. doesn't", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		otherLabels := map[string]string{"test1": "selector"}
		ignoreDk := createDynakubeWithAppInject("appMonitoring", convertToLabelSelector(otherLabels))
		notIgnoreDk := createDynakubeWithAppInject("boom", convertToLabelSelector(labels))
		notIgnoreDk.Annotations = map[string]string{
			exp.InjectionIgnoredNamespacesKey: "[\"asd\"]",
		}
		namespace := createNamespace("openshift-something", labels)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*ignoreDk, *notIgnoreDk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
		assert.Equal(t, notIgnoreDk.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
	t.Run("Double dynakube, 1. doesn't, 2. ignores openshift namespaces", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		otherLabels := map[string]string{"test1": "selector"}
		ignoreDk := createDynakubeWithAppInject("appMonitoring", convertToLabelSelector(otherLabels))
		notIgnoreDk := createDynakubeWithAppInject("boom", convertToLabelSelector(labels))
		notIgnoreDk.Annotations = map[string]string{
			exp.InjectionIgnoredNamespacesKey: "[\"asd\"]",
		}
		namespace := createNamespace("openshift-something", labels)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*notIgnoreDk, *ignoreDk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
		assert.Equal(t, notIgnoreDk.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
	t.Run("Remove stale dynakube entry for no longer matching namespace with only metadata enrichment", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		movedDk := createDynakubeWithMetadataEnrichment("moved-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: movedDk.Name,
		}
		namespace := createNamespace("test-namespace", nsLabels)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*movedDk}})
		require.NoError(t, err)
		require.True(t, updated)
		assert.Empty(t, namespace.Labels)
	})
	t.Run("Remove injection label from ignored-namespaces if present from previous setup", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(labels))
		namespace := createNamespace("test-namespace", labels)

		updated, err := updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
		require.Equal(t, namespace.Labels[dtwebhook.InjectionInstanceLabel], dk.Name)
		dk.SetAnnotations(map[string]string{exp.InjectionIgnoredNamespacesKey: "[\"" + namespace.Name + "\"]"})
		updated, err = updateNamespace(namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 1)
	})
}
