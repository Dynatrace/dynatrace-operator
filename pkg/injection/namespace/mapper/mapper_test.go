package mapper

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createBaseDynakube(name string, appInjection bool, metadataEnrichment bool) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dynatrace"},
		Spec: dynakube.DynaKubeSpec{
			MetadataEnrichment: metadataenrichment.Spec{
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

func createDynakubeWithOTLP(name string, selector metav1.LabelSelector) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dynatrace"},
		Spec: dynakube.DynaKubeSpec{
			OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
				Signals: otlp.SignalConfiguration{
					Traces: &otlp.TracesSignal{},
				},
				NamespaceSelector: selector,
			},
		},
	}

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

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
	})
	t.Run("Add to namespace, when only metadata enrichment is enabled", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithMetadataEnrichment("dk-test", convertToLabelSelector(labels))
		namespace := createNamespace("test-namespace", labels)

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
	})
	t.Run("Add to namespace, with metadata and appInjection enabled", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithMetadataAndAppInjection("appMonitoring", convertToLabelSelector(labels))

		namespace := createNamespace("test", labels)

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

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

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
		assert.Equal(t, dk.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
	t.Run("Remove stale dynakube entry for no longer matching ns", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		movedDK := createDynakubeWithAppInject("moved-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: movedDK.Name,
		}
		namespace := createNamespace("test-namespace", nsLabels)

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*movedDK}})
		require.NoError(t, err)
		require.True(t, updated)
		assert.Empty(t, namespace.Labels)
	})
	t.Run("Throw error in case of conflicting Dynakubes", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(labels))
		conflictingDK := createDynakubeWithMetadataEnrichment("conflicting-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: dk.Name,
			"test":                           "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)

		_, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*conflictingDK, *dk}})

		require.Error(t, err)
	})
	t.Run("Ignore kube namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("kube-something", nil)

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.False(t, updated)
		assert.Empty(t, namespace.Labels)
	})

	t.Run("Ignore openshift namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("openshift-something", nil)

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.False(t, updated)
		assert.Empty(t, namespace.Labels)
	})
	t.Run("Double dynakube, 1. ignores openshift namespaces, 2. doesn't", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		otherLabels := map[string]string{"test1": "selector"}
		ignoreDK := createDynakubeWithAppInject("appMonitoring", convertToLabelSelector(otherLabels))
		notIgnoreDK := createDynakubeWithAppInject("boom", convertToLabelSelector(labels))
		notIgnoreDK.Annotations = map[string]string{
			exp.InjectionIgnoredNamespacesKey: "[\"asd\"]",
		}
		namespace := createNamespace("openshift-something", labels)

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*ignoreDK, *notIgnoreDK}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
		assert.Equal(t, notIgnoreDK.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
	t.Run("Double dynakube, 1. doesn't, 2. ignores openshift namespaces", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		otherLabels := map[string]string{"test1": "selector"}
		ignoreDK := createDynakubeWithAppInject("appMonitoring", convertToLabelSelector(otherLabels))
		notIgnoreDK := createDynakubeWithAppInject("boom", convertToLabelSelector(labels))
		notIgnoreDK.Annotations = map[string]string{
			exp.InjectionIgnoredNamespacesKey: "[\"asd\"]",
		}
		namespace := createNamespace("openshift-something", labels)

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*notIgnoreDK, *ignoreDK}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
		assert.Equal(t, notIgnoreDK.Name, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	})
	t.Run("Remove stale dynakube entry for no longer matching namespace with only metadata enrichment", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		movedDK := createDynakubeWithMetadataEnrichment("moved-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: movedDK.Name,
		}
		namespace := createNamespace("test-namespace", nsLabels)

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*movedDK}})
		require.NoError(t, err)
		require.True(t, updated)
		assert.Empty(t, namespace.Labels)
	})
	t.Run("Remove injection label from ignored-namespaces if present from previous setup", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(labels))
		namespace := createNamespace("test-namespace", labels)

		updated, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 2)
		require.Equal(t, namespace.Labels[dtwebhook.InjectionInstanceLabel], dk.Name)
		dk.SetAnnotations(map[string]string{exp.InjectionIgnoredNamespacesKey: "[\"" + namespace.Name + "\"]"})
		updated, err = updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}})

		require.NoError(t, err)
		require.True(t, updated)
		assert.Len(t, namespace.Labels, 1)
	})

	t.Run("Throw error for conflicting OTLP and OneAgent DynaKubes", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		otlpDK := createDynakubeWithOTLP("otlp-dk", convertToLabelSelector(labels))
		oaDK := createDynakubeWithAppInject("oa-dk", convertToLabelSelector(labels))
		namespace := createNamespace("test-namespace", labels)

		_, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*otlpDK, *oaDK}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrorConflictingNamespace)
	})

	t.Run("Throw error for conflicting OTLP and MetadataEnrichment DynaKubes", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		otlpDK := createDynakubeWithOTLP("otlp-dk", convertToLabelSelector(labels))
		meDK := createDynakubeWithMetadataEnrichment("me-dk", convertToLabelSelector(labels))
		namespace := createNamespace("test-namespace", labels)

		_, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*otlpDK, *meDK}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrorConflictingNamespace)
	})

	t.Run("Throw error for two conflicting OTLP DynaKubes", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
		otlpDK1 := createDynakubeWithOTLP("otlp-dk-1", convertToLabelSelector(labels))
		otlpDK2 := createDynakubeWithOTLP("otlp-dk-2", convertToLabelSelector(labels))
		namespace := createNamespace("test-namespace", labels)

		_, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*otlpDK1, *otlpDK2}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrorConflictingNamespace)
	})

	t.Run("No conflict for non-overlapping OTLP DynaKubes", func(t *testing.T) {
		labels1 := map[string]string{"env": "prod"}
		labels2 := map[string]string{"env": "dev"}
		otlpDK1 := createDynakubeWithOTLP("otlp-dk-prod", convertToLabelSelector(labels1))
		otlpDK2 := createDynakubeWithOTLP("otlp-dk-dev", convertToLabelSelector(labels2))
		namespace := createNamespace("prod-namespace", labels1)

		_, err := updateNamespace(t.Context(), namespace, &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*otlpDK1, *otlpDK2}})

		require.NoError(t, err)
	})
}

func TestMapFromDynakube_MatchNamespaces(t *testing.T) {
	t.Run("AppInjection and MetadataEnrichment with same selector", func(t *testing.T) {
		labels := map[string]string{"team": "a"}
		selector := convertToLabelSelector(labels)
		dk := createDynakubeWithMetadataAndAppInjection("dk-cache", selector)

		nsList := &corev1.NamespaceList{
			Items: []corev1.Namespace{
				*createNamespace("ns-a", map[string]string{"team": "a"}),
				*createNamespace("ns-b", map[string]string{"team": "b"}),
				*createNamespace("kube-system", nil),
			},
		}

		dkList := &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}}
		dm := DynakubeMapper{dk: dk}

		_, err := dm.mapFromDynakube(nsList, dkList)
		require.NoError(t, err)

		require.Len(t, dm.matchedOANamespaces, 1)
		require.Len(t, dm.matchedMENamespaces, 1)
		assert.Equal(t, "ns-a", dm.matchedOANamespaces[0])
		assert.Equal(t, "ns-a", dm.matchedMENamespaces[0])
	})

	t.Run("OneAgent and MetadataEnrichment with different selectors", func(t *testing.T) {
		appLabels := map[string]string{"team": "a"}
		metaLabels := map[string]string{"env": "prod"}

		dk := createBaseDynakube("dk", true, true)
		dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = convertToLabelSelector(appLabels)
		dk.Spec.MetadataEnrichment.NamespaceSelector = convertToLabelSelector(metaLabels)

		nsList := &corev1.NamespaceList{
			Items: []corev1.Namespace{
				*createNamespace("ns-a", map[string]string{"team": "a", "env": "prod"}),
				*createNamespace("ns-b", map[string]string{"team": "a"}),
				*createNamespace("ns-c", map[string]string{"env": "prod"}),
				*createNamespace("ns-d", map[string]string{"team": "b"}),
			},
		}

		dkList := &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}}
		dm := DynakubeMapper{dk: dk}

		_, err := dm.mapFromDynakube(nsList, dkList)
		require.NoError(t, err)

		require.Len(t, dm.matchedOANamespaces, 2)
		require.Len(t, dm.matchedMENamespaces, 2)
		assert.Contains(t, dm.matchedOANamespaces, "ns-a")
		assert.Contains(t, dm.matchedOANamespaces, "ns-b")
		assert.Contains(t, dm.matchedMENamespaces, "ns-a")
		assert.Contains(t, dm.matchedMENamespaces, "ns-c")

		require.Len(t, dm.matchedOANamespaces, 2)
		require.Len(t, dm.matchedMENamespaces, 2)
		assert.Contains(t, dm.matchedOANamespaces, "ns-a")
		assert.Contains(t, dm.matchedOANamespaces, "ns-b")
		assert.Contains(t, dm.matchedMENamespaces, "ns-a")
		assert.Contains(t, dm.matchedMENamespaces, "ns-c")
	})

	t.Run("Only OneAgent enabled with multiple matching namespaces", func(t *testing.T) {
		labels := map[string]string{"env": "dev"}
		selector := convertToLabelSelector(labels)
		dk := createDynakubeWithAppInject("dk", selector)

		nsList := &corev1.NamespaceList{
			Items: []corev1.Namespace{
				*createNamespace("ns-dev-1", map[string]string{"env": "dev"}),
				*createNamespace("ns-dev-2", map[string]string{"env": "dev"}),
				*createNamespace("ns-prod", map[string]string{"env": "prod"}),
				*createNamespace("kube-system", nil),
			},
		}

		dkList := &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}}
		dm := DynakubeMapper{dk: dk}

		_, err := dm.mapFromDynakube(nsList, dkList)
		require.NoError(t, err)

		require.Len(t, dm.matchedOANamespaces, 2)
		require.Empty(t, dm.matchedMENamespaces)
		assert.Contains(t, dm.matchedOANamespaces, "ns-dev-1")
		assert.Contains(t, dm.matchedOANamespaces, "ns-dev-2")
	})

	t.Run("Only MetadataEnrichment enabled with multiple matching namespaces", func(t *testing.T) {
		labels := map[string]string{"monitoring": "enabled"}
		selector := convertToLabelSelector(labels)
		dk := createDynakubeWithMetadataEnrichment("dk", selector)

		nsList := &corev1.NamespaceList{
			Items: []corev1.Namespace{
				*createNamespace("ns-mon-1", map[string]string{"monitoring": "enabled"}),
				*createNamespace("ns-mon-2", map[string]string{"monitoring": "enabled"}),
				*createNamespace("ns-no-mon", map[string]string{"monitoring": "disabled"}),
				*createNamespace("ns-d", nil),
			},
		}

		dkList := &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}}
		dm := DynakubeMapper{dk: dk}

		_, err := dm.mapFromDynakube(nsList, dkList)
		require.NoError(t, err)

		require.Empty(t, dm.matchedOANamespaces)
		require.Len(t, dm.matchedMENamespaces, 2)
		assert.Contains(t, dm.matchedMENamespaces, "ns-mon-1")
		assert.Contains(t, dm.matchedMENamespaces, "ns-mon-2")
	})

	t.Run("no matching namespaces for selector", func(t *testing.T) {
		labels := map[string]string{"nonexistent": "label"}
		selector := convertToLabelSelector(labels)
		dk := createDynakubeWithMetadataAndAppInjection("dk", selector)

		nsList := &corev1.NamespaceList{
			Items: []corev1.Namespace{
				*createNamespace("ns-a", map[string]string{"team": "a"}),
				*createNamespace("ns-b", map[string]string{"env": "prod"}),
				*createNamespace("ns-c", nil),
			},
		}

		dkList := &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}}
		dm := DynakubeMapper{dk: dk}

		_, err := dm.mapFromDynakube(nsList, dkList)
		require.NoError(t, err)

		require.Empty(t, dm.matchedOANamespaces)
		require.Empty(t, dm.matchedMENamespaces)

		require.Empty(t, dm.matchedOANamespaces)
		require.Empty(t, dm.matchedMENamespaces)
	})

	t.Run("Track OTLP matched namespaces in DynakubeMapper", func(t *testing.T) {
		labels := map[string]string{"otlp": "enabled"}
		selector := convertToLabelSelector(labels)
		dk := createDynakubeWithOTLP("dk-otlp", selector)

		nsList := &corev1.NamespaceList{
			Items: []corev1.Namespace{
				*createNamespace("ns-with-otlp", labels),
				*createNamespace("ns-without-otlp", map[string]string{"other": "label"}),
			},
		}

		dkList := &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}}
		dm := DynakubeMapper{dk: dk, matchedOTLPNamespaces: []string{}}

		_, err := dm.mapFromDynakube(nsList, dkList)
		require.NoError(t, err)

		require.Len(t, dm.matchedOTLPNamespaces, 1)
		assert.Equal(t, "ns-with-otlp", dm.matchedOTLPNamespaces[0])
	})

	t.Run("OTLP with empty namespace selector matches all namespaces", func(t *testing.T) {
		// Create OTLP DynaKube with empty selector (should match all)
		dk := createDynakubeWithOTLP("dk-otlp-all", metav1.LabelSelector{})

		nsList := &corev1.NamespaceList{
			Items: []corev1.Namespace{
				*createNamespace("ns-a", map[string]string{"env": "prod"}),
				*createNamespace("ns-b", map[string]string{"env": "dev"}),
				*createNamespace("ns-c", nil),
			},
		}

		dkList := &dynakube.DynaKubeList{Items: []dynakube.DynaKube{*dk}}
		dm := DynakubeMapper{dk: dk, matchedOTLPNamespaces: []string{}}

		_, err := dm.mapFromDynakube(nsList, dkList)
		require.NoError(t, err)

		require.Len(t, dm.matchedOTLPNamespaces, 3)
		assert.Contains(t, dm.matchedOTLPNamespaces, "ns-a")
		assert.Contains(t, dm.matchedOTLPNamespaces, "ns-b")
		assert.Contains(t, dm.matchedOTLPNamespaces, "ns-c")
	})
}
