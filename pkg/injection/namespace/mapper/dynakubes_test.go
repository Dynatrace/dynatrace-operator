package mapper

import (
	"context"
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
			exp.InjectionIgnoredNamespacesKey: "[]",
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
	t.Run("Remove "+consts.BootstrapperInitSecretName+", "+consts.BootstrapperInitCertsSecretName+" and "+consts.OTLPExporterSecretName+" secrets"+" and "+consts.OTLPExporterCertsSecretName+" secrets", func(t *testing.T) {
		clt := fake.NewClient(namespace, namespace2)
		ctx := context.Background()

		namespaces, err := GetNamespacesForDynakube(ctx, clt, dk.Name)
		require.NoError(t, err)

		clt.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: consts.BootstrapperInitSecretName, Namespace: namespace.Name}})
		clt.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: consts.BootstrapperInitCertsSecretName, Namespace: namespace.Name}})
		clt.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: consts.OTLPExporterSecretName, Namespace: namespace.Name}})

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

		dkNodeImagePull := createDynakubeWithNodeImagePullAndNoCSI("dk-test", convertToLabelSelector(labels))

		labels := map[string]string{
			dtwebhook.InjectionInstanceLabel: dkNodeImagePull.Name,
		}

		ns := createNamespace("ns-bootstrapper", labels)
		ns2 := createNamespace("ns-bootstrapper2", labels)

		clt := fake.NewClient(ns, ns2)
		ctx := context.Background()

		namespaces, err := GetNamespacesForDynakube(ctx, clt, dkNodeImagePull.Name)
		require.NoError(t, err)

		var secretNS1 corev1.Secret

		clt.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: consts.BootstrapperInitSecretName, Namespace: ns.Name}})

		err = clt.Get(ctx, types.NamespacedName{Name: consts.BootstrapperInitSecretName, Namespace: ns.Name}, &secretNS1)
		require.NoError(t, err)

		require.NotEmpty(t, secretNS1)
		assert.Equal(t, consts.BootstrapperInitSecretName, secretNS1.Name)

		var secretNS2 corev1.Secret

		clt.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: consts.BootstrapperInitSecretName, Namespace: ns2.Name}})

		err = clt.Get(ctx, types.NamespacedName{Name: consts.BootstrapperInitSecretName, Namespace: ns2.Name}, &secretNS2)
		require.NoError(t, err)

		require.NotEmpty(t, secretNS2)
		assert.Equal(t, consts.BootstrapperInitSecretName, secretNS2.Name)

		dm := NewDynakubeMapper(ctx, clt, clt, "dynatrace", dkNodeImagePull)
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
