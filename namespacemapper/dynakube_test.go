package namespacemapper

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

func TestMapFromDynaKubeCodeModules(t *testing.T) {
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
	t.Run("Add to new config map", func(t *testing.T) {
		clt := fake.NewClient(dk, namespace)

		err := MapFromDynaKubeCodeModules(context.TODO(), clt, "dynatrace", dk)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: CodeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})
	t.Run("Add to existing config map", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: CodeModulesMapName, Namespace: "dynatrace"},
			Data:       map[string]string{"other-ns": "other-dk"},
		}
		clt := fake.NewClient(dk, &oldCfg, namespace)

		err := MapFromDynaKubeCodeModules(context.TODO(), clt, "dynatrace", dk)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: CodeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(cfg.Data))
	})
	t.Run("Overwrite stale entry in config map", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: CodeModulesMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: "other-dk"},
		}
		clt := fake.NewClient(dk, &oldCfg, namespace)

		err := MapFromDynaKubeCodeModules(context.TODO(), clt, "dynatrace", dk)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: CodeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})
	t.Run("Remove stale dynakube entry", func(t *testing.T) {
		movedDk := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		}
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: CodeModulesMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: movedDk.Name},
		}
		clt := fake.NewClient(&oldCfg, movedDk, namespace)

		err := MapFromDynaKubeCodeModules(context.TODO(), clt, "dynatrace", movedDk)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: CodeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cfg.Data))
	})
}

func TestMapFromDynaKubeDataIngest(t *testing.T) {
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
	t.Run("Add to new config map", func(t *testing.T) {
		clt := fake.NewClient(dk, namespace)

		err := MapFromDynakubeDataIngest(context.TODO(), clt, "dynatrace", dk)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: DataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})
	t.Run("Add to existing config map", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: DataIngestMapName, Namespace: "dynatrace"},
			Data:       map[string]string{"other-ns": "other-dk"},
		}
		clt := fake.NewClient(dk, &oldCfg, namespace)

		err := MapFromDynakubeDataIngest(context.TODO(), clt, "dynatrace", dk)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: DataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(cfg.Data))
	})
	t.Run("Overwrite stale entry in config map", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: DataIngestMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: "other-dk"},
		}
		clt := fake.NewClient(dk, &oldCfg, namespace)

		err := MapFromDynakubeDataIngest(context.TODO(), clt, "dynatrace", dk)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: DataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})
	t.Run("Remove stale dynakube entry", func(t *testing.T) {
		movedDk := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "dataIngest-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
					CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
						Enabled: true,
					},
				},
			},
		}
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: DataIngestMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: movedDk.Name},
		}
		clt := fake.NewClient(&oldCfg, movedDk, namespace)

		err := MapFromDynakubeDataIngest(context.TODO(), clt, "dynatrace", movedDk)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: DataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cfg.Data))
	})
}

func TestUnmapFromDynaKube(t *testing.T) {
	cfgDI := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: DataIngestMapName, Namespace: "dynatrace"},
		Data:       map[string]string{"ns1": "dk", "ns2": "dk", "ns3": "other-dk"},
	}
	cfgCM := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: CodeModulesMapName, Namespace: "dynatrace"},
		Data:       map[string]string{"ns1": "dk", "ns3": "other-dk"},
	}
	t.Run("Remove from empty => no error", func(t *testing.T) {
		clt := fake.NewClient()
		err := UnmapFromDynaKube(context.TODO(), clt, "dynatrace", "dk")
		assert.NoError(t, err)
	})
	t.Run("Remove from everywhere, multiple entries", func(t *testing.T) {
		clt := fake.NewClient(&cfgDI, &cfgCM)
		err := UnmapFromDynaKube(context.TODO(), clt, "dynatrace", "dk")
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: DataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
		err = clt.Get(context.TODO(), types.NamespacedName{Name: CodeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})
}
