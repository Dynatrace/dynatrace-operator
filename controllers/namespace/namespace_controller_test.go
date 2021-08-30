package namespace

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	mapper "github.com/Dynatrace/dynatrace-operator/namespacemapper"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileCM(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			CodeModules: dynatracev1alpha1.CodeModulesSpec{
				Enabled:           true,
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: labels},
			},
		},
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-namespace",
			Labels: labels,
		},
	}

	reconciler := ReconcileNamespaces{
		namespace: "dynatrace",
		logger:    log.Log.WithName("namespace.controller"),
	}

	t.Run("Add to new config map", func(t *testing.T) {
		clt := fake.NewClient(dk, namespace)
		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: mapper.CodeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})

	t.Run("Error, 2 dynakube point to same namespace", func(t *testing.T) {
		dk2 := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled:           true,
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
			},
		}
		clt := fake.NewClient(dk, dk2, namespace)

		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.Error(t, err)
	})

	t.Run("Add to existing config map", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: mapper.CodeModulesMapName, Namespace: "dynatrace"},
			Data:       map[string]string{"other-ns": "other-dk"},
		}
		clt := fake.NewClient(dk, &oldCfg, namespace)

		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: mapper.CodeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(cfg.Data))
	})

	t.Run("Remove stale namespace entry - no dynakube", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: mapper.CodeModulesMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: "deleted-dk"},
		}
		clt := fake.NewClient(&oldCfg, namespace)

		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: mapper.CodeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cfg.Data))
	})

	t.Run("Remove stale namespace entry - no namespace", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: mapper.CodeModulesMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: "dk"},
		}
		clt := fake.NewClient(&oldCfg)

		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: mapper.CodeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cfg.Data))
	})
}

func TestReconcileDI(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "dataIngest-1", Namespace: "dynatrace"},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
				CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
					Enabled: true,
				},
				Selector: &metav1.LabelSelector{MatchLabels: labels},
			},
		},
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-namespace",
			Labels: labels,
		},
	}
	reconciler := ReconcileNamespaces{
		namespace: "dynatrace",
		logger:    log.Log.WithName("namespace.controller"),
	}
	t.Run("Add to new config map", func(t *testing.T) {
		clt := fake.NewClient(dk, namespace)

		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: mapper.DataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})

	t.Run("Error, 2 dynakube point to same namespace", func(t *testing.T) {
		dk2 := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "dataIngest-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
					CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
						Enabled: true,
					},
					Selector: &metav1.LabelSelector{MatchLabels: labels},
				},
			},
		}
		clt := fake.NewClient(dk, dk2, namespace)

		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.Error(t, err)
	})

	t.Run("Add to existing config map", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: mapper.DataIngestMapName, Namespace: "dynatrace"},
			Data:       map[string]string{"other-ns": "other-dk"},
		}
		clt := fake.NewClient(dk, &oldCfg, namespace)

		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: mapper.DataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(cfg.Data))
	})

	t.Run("Remove stale namespace entry - no dynakube", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: mapper.DataIngestMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: "deleted-dk"},
		}
		clt := fake.NewClient(&oldCfg, namespace)

		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: mapper.DataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cfg.Data))
	})
	t.Run("Remove stale namespace entry - no namespace", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: mapper.DataIngestMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: "dk"},
		}
		clt := fake.NewClient(&oldCfg)

		reconciler.client = clt
		reconciler.apiReader = clt

		_, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: mapper.DataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cfg.Data))
	})
}
