package mapper

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestMapFromDynakube(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "dk-test", Namespace: "dynatrace"},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			MonitoredNamespaces: &metav1.LabelSelector{MatchLabels: labels},
			CodeModules: dynatracev1alpha1.CodeModulesSpec{
				Enabled: true,
			},
			DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
				CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
					Enabled: true,
				},
			},
		},
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-namespace",
			Labels: labels,
		},
	}
	t.Run("Add to namespace", func(t *testing.T) {
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk)
		err := dm.MapFromDynakube()
		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(ns.Labels))
		assert.Equal(t, 1, len(ns.Annotations))
	})
	t.Run("Overwrite stale entry in annotations", func(t *testing.T) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					InstanceLabel: "old-dk",
					"test":        "selector",
				},
			},
		}
		clt := fake.NewClient(dk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", dk)
		err := dm.MapFromDynakube()
		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(ns.Labels))
		assert.Equal(t, 1, len(ns.Annotations))
	})
	t.Run("Remove stale dynakube entry for no longer matching ns", func(t *testing.T) {
		movedDk := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "moved-dk", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
				DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
					CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
						Enabled: true,
					},
				},
			},
		}
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					InstanceLabel: movedDk.Name,
				},
			},
		}
		clt := fake.NewClient(movedDk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", movedDk)
		err := dm.MapFromDynakube()
		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(ns.Annotations))
	})
	t.Run("Throw error in case of conflicting Dynakubes", func(t *testing.T) {
		conflictingDk := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "conflicting-dk", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				MonitoredNamespaces: &metav1.LabelSelector{MatchLabels: labels},
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
				DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
					CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
						Enabled: true,
					},
				},
			},
		}
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					InstanceLabel: dk.Name,
					"test":        "selector",
				},
			},
		}
		clt := fake.NewClient(dk, conflictingDk, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", conflictingDk)
		err := dm.MapFromDynakube()
		assert.Error(t, err)
	})
	t.Run("Allow multiple dynakubes with different features", func(t *testing.T) {
		differentDk1 := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "dk1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				MonitoredNamespaces: &metav1.LabelSelector{MatchLabels: labels},
				DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
					CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
						Enabled: true,
					},
				},
			},
		}
		differentDk2 := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "dk2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				MonitoredNamespaces: &metav1.LabelSelector{MatchLabels: labels},
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		}
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					InstanceLabel: dk.Name,
					"test":        "selector",
				},
			},
		}
		clt := fake.NewClient(differentDk1, differentDk2, namespace)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", differentDk1)
		err := dm.MapFromDynakube()
		assert.NoError(t, err)
		var ns corev1.Namespace
		err = clt.Get(context.TODO(), types.NamespacedName{Name: namespace.Name}, &ns)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(ns.Labels))
		assert.Equal(t, 1, len(ns.Annotations))
	})
}

func TestUnmapFromDynaKube(t *testing.T) {
	dk := dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "dk"},
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				InstanceLabel: dk.Name,
			},
		},
	}
	namespace2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace2",
			Labels: map[string]string{
				InstanceLabel: dk.Name,
			},
		},
	}
	t.Run("Remove from no ns => no error", func(t *testing.T) {
		clt := fake.NewClient()
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", &dk)
		err := dm.UnmapFromDynaKube()
		assert.NoError(t, err)
	})
	t.Run("Remove from everywhere, multiple entries", func(t *testing.T) {
		clt := fake.NewClient(namespace, namespace2)
		dm := NewDynakubeMapper(context.TODO(), clt, clt, "dynatrace", &dk)
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
