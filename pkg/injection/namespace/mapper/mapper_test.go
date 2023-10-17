package mapper

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestDynakubeWithAppInject(name string, labels map[string]string, labelExpression []metav1.LabelSelectorRequirement) *dynatracev1beta1.DynaKube {
	dk := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dynatrace"},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
			},
		},
	}
	if labels != nil {
		dk.Spec.NamespaceSelector = metav1.LabelSelector{MatchLabels: labels}
	}
	if labelExpression != nil {
		dk.Spec.NamespaceSelector = metav1.LabelSelector{MatchExpressions: labelExpression}
	}
	return dk
}

func createTestDynakubeWithMultipleFeatures(name string, labels map[string]string) *dynatracev1beta1.DynaKube {
	dk := createTestDynakubeWithAppInject(name, labels, nil)
	dk.Spec.Routing.Enabled = true
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

func TestUpdateNamespace(t *testing.T) {
	t.Run("Add to namespace", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createTestDynakubeWithMultipleFeatures("dk-test", labels)
		namespace := createNamespace("test-namespace", labels)

		updated, err := updateNamespace(namespace, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Equal(t, 2, len(namespace.Labels))
	})
	t.Run("Overwrite stale entry in labels", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createTestDynakubeWithMultipleFeatures("dk-test", labels)
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: "old-dk",
			"test":                           "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)

		updated, err := updateNamespace(namespace, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Equal(t, 2, len(namespace.Labels))
		assert.Equal(t, dk.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
	t.Run("Remove stale dynakube entry for no longer matching ns", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		movedDk := createTestDynakubeWithAppInject("moved-dk", labels, nil)
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: movedDk.Name,
		}
		namespace := createNamespace("test-namespace", nsLabels)

		updated, err := updateNamespace(namespace, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{*movedDk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Equal(t, 0, len(namespace.Labels))
	})
	t.Run("Throw error in case of conflicting Dynakubes", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createTestDynakubeWithMultipleFeatures("dk-test", labels)
		conflictingDk := createTestDynakubeWithMultipleFeatures("conflicting-dk", labels)
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: dk.Name,
			"test":                           "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)

		_, err := updateNamespace(namespace, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{*conflictingDk, *dk}})

		assert.Error(t, err)
	})
	t.Run("Ignore kube namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil)
		namespace := createNamespace("kube-something", nil)

		updated, err := updateNamespace(namespace, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{*dk}})

		require.NoError(t, err)
		require.False(t, updated)
		assert.Equal(t, 0, len(namespace.Labels))
	})

	t.Run("Ignore openshift namespaces", func(t *testing.T) {
		dk := createTestDynakubeWithMultipleFeatures("appMonitoring", nil)
		namespace := createNamespace("openshift-something", nil)

		updated, err := updateNamespace(namespace, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{*dk}})

		require.NoError(t, err)
		require.False(t, updated)
		assert.Equal(t, 0, len(namespace.Labels))
	})
	t.Run("Double dynakube, 1. ignores openshift namespaces, 2. doesn't", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		otherLabels := map[string]string{"test1": "selector"}
		ignoreDk := createTestDynakubeWithMultipleFeatures("appMonitoring", otherLabels)
		notIgnoreDk := createTestDynakubeWithMultipleFeatures("boom", labels)
		notIgnoreDk.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureIgnoredNamespaces: "[\"asd\"]",
		}
		namespace := createNamespace("openshift-something", labels)

		updated, err := updateNamespace(namespace, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{*ignoreDk, *notIgnoreDk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Equal(t, 2, len(namespace.Labels))
		assert.Equal(t, notIgnoreDk.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
	t.Run("Double dynakube, 1. doesn't, 2. ignores openshift namespaces", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		otherLabels := map[string]string{"test1": "selector"}
		ignoreDk := createTestDynakubeWithMultipleFeatures("appMonitoring", otherLabels)
		notIgnoreDk := createTestDynakubeWithMultipleFeatures("boom", labels)
		notIgnoreDk.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureIgnoredNamespaces: "[\"asd\"]",
		}
		namespace := createNamespace("openshift-something", labels)

		updated, err := updateNamespace(namespace, &dynatracev1beta1.DynaKubeList{Items: []dynatracev1beta1.DynaKube{*notIgnoreDk, *ignoreDk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Equal(t, 2, len(namespace.Labels))
		assert.Equal(t, notIgnoreDk.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
}
