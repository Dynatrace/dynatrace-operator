// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package mapper

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestMapFromDynakube_InvalidSelectorIsolatesBlastRadius guards the resolved product decision:
// a malformed selector on one feature must not suppress another, independently-configured feature's otherwise-valid matches for the same shared secret.
func TestMapFromDynakube_InvalidSelectorIsolatesBlastRadius(t *testing.T) {
	invalidSelector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "team", Operator: "NotARealOperator", Values: []string{"a"}},
		},
	}
	validSelector := convertToLabelSelector(map[string]string{"team": "a"})

	dk := createBaseDynakube("dk-isolation", true, true)
	dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = invalidSelector
	dk.Spec.MetadataEnrichment.NamespaceSelector = validSelector

	nsA := createNamespace("ns-a", map[string]string{"team": "a"})

	clt := fake.NewClient(dk, nsA)
	dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", dk)

	err := dm.MapFromDynakube()
	require.NoError(t, err)

	assert.Empty(t, dm.namespaceNamesFor(flagOneAgent))
	assert.Equal(t, []string{"ns-a"}, dm.namespaceNamesFor(flagMetadata))

	assert.True(t, dm.invalidSelectors.isOneAgent())
	assert.False(t, dm.invalidSelectors.isMetadata())

	oaCondition := meta.FindStatusCondition(*dk.Conditions(), oneAgentNamespacesMonitoredConditionType.String())
	require.NotNil(t, oaCondition)
	assert.Equal(t, invalidSelectorReason, oaCondition.Reason)
	assert.Equal(t, metav1.ConditionFalse, oaCondition.Status)

	meCondition := meta.FindStatusCondition(*dk.Conditions(), metadataEnrichmentNamespacesMonitoredConditionType.String())
	require.NotNil(t, meCondition)
	assert.Equal(t, matchesFoundReason, meCondition.Reason)
	assert.Equal(t, metav1.ConditionTrue, meCondition.Status)

	var ns corev1.Namespace
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: "ns-a"}, &ns))
	assert.Equal(t, dk.Name, ns.Labels[dtwebhook.InjectionInstanceLabel])
}

func TestMapFromDynakube(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(labels))
	namespace := createNamespace("test-namespace", labels)

	t.Run("Add to namespace", func(t *testing.T) {
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(t.Context(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Len(t, ns.Labels, 2)
	})
	t.Run("Overwrite stale entry in labels", func(t *testing.T) {
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: "old-dk",
			"test":                           "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(t.Context(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Len(t, ns.Labels, 2)
	})
	t.Run("Remove stale dynakube entry for no longer matching ns", func(t *testing.T) {
		movedDK := createDynakubeWithAppInject("moved-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: movedDK.Name,
		}
		namespace := createNamespace("test-namespace", nsLabels)
		clt := fake.NewClient(movedDK, namespace)
		dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", movedDK)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(t.Context(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
	})
	t.Run("Throw error in case of conflicting Dynakubes", func(t *testing.T) {
		conflictingDK := createDynakubeWithAppInject("conflicting-dk", convertToLabelSelector(labels))
		nsLabels := map[string]string{
			dtwebhook.InjectionInstanceLabel: dk.Name,
			"test":                           "selector",
		}
		namespace := createNamespace("test-namespace", nsLabels)
		clt := fake.NewClient(dk, conflictingDK, namespace)
		dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", conflictingDK)

		err := dm.MapFromDynakube()

		require.Error(t, err)
	})
	t.Run("Ignore kube namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("kube-something", nil)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(t.Context(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
	})

	t.Run("Ignore openshift namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(t.Context(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
	})
	t.Run("ComponentFeature flag for monitoring system namespaces", func(t *testing.T) {
		dk := createDynakubeWithAppInject("appMonitoring", metav1.LabelSelector{})
		dk.Annotations = map[string]string{
			exp.InjectionIgnoredNamespacesKey: "[]",
		}
		namespace := createNamespace("openshift-something", nil)
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", dk)

		err := dm.MapFromDynakube()

		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(t.Context(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Len(t, ns.Labels, 1)
	})

	t.Run("namespace no longer matches selector => secrets deleted", func(t *testing.T) {
		matchingLabels := map[string]string{"foo": "bar"}
		dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(matchingLabels))

		namespace := createNamespace(
			"definitely-not-selected",
			map[string]string{
				dtwebhook.InjectionInstanceLabel: dk.Name,
			},
		)

		clt := fake.NewClient(dk, namespace)
		ctx := t.Context()

		secretNames := []string{
			consts.BootstrapperInitSecretName,
			consts.BootstrapperInitCertsSecretName,
			consts.OTLPExporterSecretName,
			consts.OTLPExporterCertsSecretName,
		}
		for _, secretName := range secretNames {
			createSecret(t, clt, secretName, namespace.Name)
		}

		dm := NewDynakubeMapper(ctx, clt, clt, "dynatrace", dk)
		err := dm.MapFromDynakube()
		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(ctx, types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)

		for _, secretName := range secretNames {
			var secret corev1.Secret
			err = clt.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace.Name}, &secret)
			assert.Truef(t, k8serrors.IsNotFound(err), "expected secret %s to be deleted from deselected namespace", secretName)
		}
	})
}

func TestMatchingNamespaces(t *testing.T) {
	t.Run("validating webhook run on deselected ns => replicated secrets not deleted", func(t *testing.T) {
		matchingLabels := map[string]string{"foo": "bar"}
		dk := createDynakubeWithAppInject("dk-test", convertToLabelSelector(matchingLabels))

		namespace := createNamespace(
			"definitely-not-selected",
			map[string]string{
				dtwebhook.InjectionInstanceLabel: dk.Name,
			},
		)

		clt := fake.NewClient(dk, namespace)
		ctx := t.Context()

		secretNames := []string{
			consts.BootstrapperInitSecretName,
			consts.BootstrapperInitCertsSecretName,
			consts.OTLPExporterSecretName,
			consts.OTLPExporterCertsSecretName,
		}
		for _, secretName := range secretNames {
			createSecret(t, clt, secretName, namespace.Name)
		}

		// pass nil write client, just like validating webhooks
		dm := NewDynakubeMapper(ctx, nil, clt, "dynatrace", dk)

		require.NotPanics(t, func() {
			_, err := dm.MatchingNamespaces()
			require.NoError(t, err)
		})

		for _, secretName := range secretNames {
			var secret corev1.Secret
			err := clt.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace.Name}, &secret)
			require.NoErrorf(t, err, "secret %s must not be deleted when validating webhooks are run", secretName)
		}
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

		namespaces, err := GetNamespacesForDynakube(t.Context(), clt, dk.Name)
		require.NoError(t, err)

		dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", dk)
		err = dm.UnmapFromDynaKube(namespaces)
		require.NoError(t, err)
	})
	t.Run("Remove from everywhere, multiple entries", func(t *testing.T) {
		clt := fake.NewClient(namespace, namespace2)

		namespaces, err := GetNamespacesForDynakube(t.Context(), clt, dk.Name)
		require.NoError(t, err)

		dm := NewDynakubeMapper(t.Context(), clt, clt, "dynatrace", dk)
		err = dm.UnmapFromDynaKube(namespaces)
		require.NoError(t, err)

		var ns corev1.Namespace
		err = clt.Get(t.Context(), types.NamespacedName{Name: namespace.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
		err = clt.Get(t.Context(), types.NamespacedName{Name: namespace2.Name}, &ns)
		require.NoError(t, err)
		assert.Empty(t, ns.Labels)
	})
	t.Run("Remove "+consts.BootstrapperInitSecretName+", "+consts.BootstrapperInitCertsSecretName+" and "+consts.OTLPExporterSecretName+" secrets"+" and "+consts.OTLPExporterCertsSecretName+" secrets", func(t *testing.T) {
		clt := fake.NewClient(namespace, namespace2)
		ctx := t.Context()

		namespaces, err := GetNamespacesForDynakube(ctx, clt, dk.Name)
		require.NoError(t, err)

		createSecret(t, clt, consts.BootstrapperInitSecretName, namespace.Name)
		createSecret(t, clt, consts.BootstrapperInitCertsSecretName, namespace.Name)
		createSecret(t, clt, consts.OTLPExporterSecretName, namespace.Name)

		dm := NewDynakubeMapper(ctx, clt, clt, "dynatrace", dk)
		err = dm.UnmapFromDynaKube(namespaces)
		require.NoError(t, err)

		var secret corev1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: consts.BootstrapperInitSecretName, Namespace: namespace.Name}, &secret)
		assert.True(t, k8serrors.IsNotFound(err))
		err = clt.Get(ctx, types.NamespacedName{Name: consts.BootstrapperInitCertsSecretName, Namespace: namespace.Name}, &secret)
		assert.True(t, k8serrors.IsNotFound(err))
		err = clt.Get(ctx, types.NamespacedName{Name: consts.OTLPExporterSecretName, Namespace: namespace.Name}, &secret)
		assert.True(t, k8serrors.IsNotFound(err))
		err = clt.Get(ctx, types.NamespacedName{Name: consts.OTLPExporterCertsSecretName, Namespace: namespace.Name}, &secret)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("Remove "+consts.BootstrapperInitSecretName, func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		dkAppmonImage := createDynakubeWithAppInjectImage("dk-test", convertToLabelSelector(labels))

		labels := map[string]string{
			dtwebhook.InjectionInstanceLabel: dkAppmonImage.Name,
		}

		ns := createNamespace("ns-bootstrapper", labels)
		ns2 := createNamespace("ns-bootstrapper2", labels)

		clt := fake.NewClient(ns, ns2)
		ctx := t.Context()

		namespaces, err := GetNamespacesForDynakube(ctx, clt, dkAppmonImage.Name)
		require.NoError(t, err)

		var secretNS1 corev1.Secret

		createSecret(t, clt, consts.BootstrapperInitSecretName, ns.Name)

		err = clt.Get(ctx, types.NamespacedName{Name: consts.BootstrapperInitSecretName, Namespace: ns.Name}, &secretNS1)
		require.NoError(t, err)

		require.NotEmpty(t, secretNS1)
		assert.Equal(t, consts.BootstrapperInitSecretName, secretNS1.Name)

		var secretNS2 corev1.Secret

		createSecret(t, clt, consts.BootstrapperInitSecretName, ns2.Name)

		err = clt.Get(ctx, types.NamespacedName{Name: consts.BootstrapperInitSecretName, Namespace: ns2.Name}, &secretNS2)
		require.NoError(t, err)

		require.NotEmpty(t, secretNS2)
		assert.Equal(t, consts.BootstrapperInitSecretName, secretNS2.Name)

		dm := NewDynakubeMapper(ctx, clt, clt, "dynatrace", dkAppmonImage)
		err = dm.UnmapFromDynaKube(namespaces)
		require.NoError(t, err)

		var deletedSecretNS1 corev1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: consts.BootstrapperInitSecretName, Namespace: ns.Name}, &deletedSecretNS1)

		require.Empty(t, deletedSecretNS1)
		assert.NotEqual(t, consts.BootstrapperInitSecretName, deletedSecretNS1.Name)
		assert.True(t, k8serrors.IsNotFound(err))

		var deletedSecretNS2 corev1.Secret
		err = clt.Get(ctx, types.NamespacedName{Name: consts.BootstrapperInitSecretName, Namespace: ns2.Name}, &deletedSecretNS2)

		require.Empty(t, deletedSecretNS2)
		assert.NotEqual(t, consts.BootstrapperInitSecretName, deletedSecretNS2.Name)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func createSecret(t *testing.T, c client.Client, name, namespace string) {
	t.Helper()
	c.Create(t.Context(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
}
