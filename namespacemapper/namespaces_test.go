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

func TestMatchForNamespace(t *testing.T) {
	dynakubes := []dynatracev1alpha1.DynaKube{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"inject": "true",
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
					Selector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: "type", Operator: metav1.LabelSelectorOpIn, Values: []string{"server", "app"}},
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: false,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"inject": "true",
						},
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: "type", Operator: metav1.LabelSelectorOpIn, Values: []string{"server", "app"}},
						},
					},
				},
			},
		},
	}

	t.Run(`Match nothing to unlabeled namespace`, func(t *testing.T) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		}

		dynakube, err := matchForNamespace(dynakubes, namespace, func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.Selector
		})
		assert.NoError(t, err)
		assert.Nil(t, dynakube)
	})

	t.Run(`Match namespace with labels`, func(t *testing.T) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					"inject": "true",
				},
			},
		}

		dynakube, err := matchForNamespace(dynakubes, namespace, func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.Selector
		})
		assert.NoError(t, err)
		assert.NotNil(t, dynakube)
	})

	t.Run(`Match namespace with expressions`, func(t *testing.T) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					"type": "app",
				},
			},
		}

		dynakube, err := matchForNamespace(dynakubes, namespace, func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.Selector
		})
		assert.NoError(t, err)
		assert.NotNil(t, dynakube)
	})

	t.Run(`Error on multiple Dynakube matches`, func(t *testing.T) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					"type":   "app",
					"inject": "true",
				},
			},
		}

		dynakube, err := matchForNamespace(dynakubes, namespace, func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.Selector
		})
		assert.Error(t, err)
		assert.Nil(t, dynakube)
	})
}

func TestMatchForNamespaceNothingEverything(t *testing.T) {
	dynakubes := []dynatracev1alpha1.DynaKube{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
					// no 'Selector:' field means match nothing
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled:  true,
					Selector: &metav1.LabelSelector{}, // empty 'Selector:' field means match everything
				},
			},
		},
	}

	t.Run(`Match to unlabeled namespace`, func(t *testing.T) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		}

		dynakube, err := matchForNamespace(dynakubes, namespace, func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.Selector
		})
		assert.NoError(t, err)
		assert.NotNil(t, dynakube)
		assert.Equal(t, dynakube.Name, "codeModules-2")
	})
}

func TestFindCodeModules(t *testing.T) {
	instances := []dynatracev1alpha1.DynaKube{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: false,
				},
			},
		},
	}
	clt := fake.NewClient(
		&instances[0],
		&instances[1],
		&instances[2])

	codeModules, err := findDynaKubes(context.TODO(), clt, func(dk dynatracev1alpha1.DynaKube) bool {
		return dk.Spec.CodeModules.Enabled
	})
	assert.NoError(t, err)
	assert.NotNil(t, codeModules)
	assert.Equal(t, 2, len(codeModules))
}

func TestFindDataIngest(t *testing.T) {
	instances := []dynatracev1alpha1.DynaKube{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "dataIngest-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
					CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
						Enabled: true,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "dataIngest-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
					CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
						Enabled: true,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				DataIngestSpec: dynatracev1alpha1.DataIngestSpec{
					CapabilityProperties: dynatracev1alpha1.CapabilityProperties{
						Enabled: false,
					},
				},
			},
		},
	}
	clt := fake.NewClient(
		&instances[0],
		&instances[1],
		&instances[2])

	dataIngest, err := findDynaKubes(context.TODO(), clt, func(dk dynatracev1alpha1.DynaKube) bool {
		return dk.Spec.DataIngestSpec.Enabled
	})
	assert.NoError(t, err)
	assert.NotNil(t, dataIngest)
	assert.Equal(t, 2, len(dataIngest))
}

func TestMapFromNamespaceCodeModules(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			CodeModules: dynatracev1alpha1.CodeModulesSpec{
				Enabled:  true,
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

		err := mapFromNamespaceCodeModules(context.TODO(), clt, "dynatrace", *namespace)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: codeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})

	t.Run("Error, 2 dynakube point to same namespace", func(t *testing.T) {
		dk2 := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled:  true,
					Selector: &metav1.LabelSelector{MatchLabels: labels},
				},
			},
		}
		clt := fake.NewClient(dk, dk2, namespace)

		err := mapFromNamespaceCodeModules(context.TODO(), clt, "dynatrace", *namespace)
		assert.Error(t, err)
	})

	t.Run("Add to existing config map", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: codeModulesMapName, Namespace: "dynatrace"},
			Data:       map[string]string{"other-ns": "other-dk"},
		}
		clt := fake.NewClient(dk, &oldCfg, namespace)

		err := mapFromNamespaceCodeModules(context.TODO(), clt, "dynatrace", *namespace)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: codeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(cfg.Data))
	})

	t.Run("Remove stale namespace entry", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: codeModulesMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: "deleted-dk"},
		}
		clt := fake.NewClient(&oldCfg, namespace)

		err := mapFromNamespaceCodeModules(context.TODO(), clt, "dynatrace", *namespace)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: codeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cfg.Data))
	})
}

func TestMapFromNamespaceDataIngest(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
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

		err := mapFromNamespaceDataIngest(context.TODO(), clt, "dynatrace", *namespace)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: dataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})

	t.Run("Error, 2 dynakube point to same namespace", func(t *testing.T) {
		dk2 := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
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

		err := mapFromNamespaceDataIngest(context.TODO(), clt, "dynatrace", *namespace)
		assert.Error(t, err)
	})

	t.Run("Add to existing config map", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: dataIngestMapName, Namespace: "dynatrace"},
			Data:       map[string]string{"other-ns": "other-dk"},
		}
		clt := fake.NewClient(dk, &oldCfg, namespace)

		err := mapFromNamespaceDataIngest(context.TODO(), clt, "dynatrace", *namespace)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: dataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(cfg.Data))
	})

	t.Run("Remove stale namespace entry", func(t *testing.T) {
		oldCfg := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: dataIngestMapName, Namespace: "dynatrace"},
			Data:       map[string]string{namespace.Name: "deleted-dk"},
		}
		clt := fake.NewClient(&oldCfg, namespace)

		err := mapFromNamespaceDataIngest(context.TODO(), clt, "dynatrace", *namespace)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: dataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(cfg.Data))
	})
}

func TestUnmapFromNamespace(t *testing.T) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	cfgDI := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: dataIngestMapName, Namespace: "dynatrace"},
		Data:       map[string]string{namespace.Name: "dk", "other-ns": "other-dk"},
	}
	cfgCM := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: codeModulesMapName, Namespace: "dynatrace"},
		Data:       map[string]string{namespace.Name: "dk", "other-ns": "other-dk"},
	}
	clt := fake.NewClient(&cfgDI, &cfgCM, namespace)
	t.Run("Unmap from all", func(t *testing.T) {
		err := UnmapFromNamespace(context.TODO(), clt, "dynatrace", namespace.Name)
		assert.NoError(t, err)
		var cfg corev1.ConfigMap
		err = clt.Get(context.TODO(), types.NamespacedName{Name: dataIngestMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
		err = clt.Get(context.TODO(), types.NamespacedName{Name: codeModulesMapName, Namespace: "dynatrace"}, &cfg)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Data))
	})
}
