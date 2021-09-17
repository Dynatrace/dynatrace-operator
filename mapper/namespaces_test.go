package mapper

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFindDynakubeForNamespace(t *testing.T) {
	dynakubes := []*dynatracev1alpha1.DynaKube{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				MonitoredNamespaces: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"inject": "true",
					},
				},
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				MonitoredNamespaces: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{Key: "type", Operator: metav1.LabelSelectorOpIn, Values: []string{"server", "app"}},
					},
				},
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				MonitoredNamespaces: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"inject": "true",
					},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{Key: "type", Operator: metav1.LabelSelectorOpIn, Values: []string{"server", "app"}},
					},
				},
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: false,
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
		clt := fake.NewClient(dynakubes[0], dynakubes[1], dynakubes[2])
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())
		updated, err := nm.updateNamespace()
		assert.NoError(t, err)
		assert.False(t, updated)
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

		clt := fake.NewClient(dynakubes[0], dynakubes[1], dynakubes[2])
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())
		updated, err := nm.updateNamespace()
		assert.NoError(t, err)
		assert.True(t, updated)
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

		clt := fake.NewClient(dynakubes[0], dynakubes[1], dynakubes[2])
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())
		updated, err := nm.updateNamespace()
		assert.NoError(t, err)
		assert.True(t, updated)
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

		clt := fake.NewClient(dynakubes[0], dynakubes[1], dynakubes[2])
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())
		_, err := nm.updateNamespace()
		assert.Error(t, err)
	})
}

func TestMatchForNamespaceNothingEverything(t *testing.T) {
	dynakubes := []*dynatracev1alpha1.DynaKube{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				// no 'MonitoredNamespaces:' field means match everything
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				MonitoredNamespaces: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"type":   "app",
						"inject": "true",
					}},
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
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

		clt := fake.NewClient(dynakubes[0], dynakubes[1])
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())
		dynakube, err := nm.updateNamespace()
		assert.NoError(t, err)
		assert.NotNil(t, dynakube)
		//assert.Equal(t, dynakube.Name, "codeModules-1")
	})
}

func TestMapFromNamespace(t *testing.T) {
	labels := map[string]string{"test": "selector"}
	dk := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
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
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()
		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, 2, len(nm.targetNs.Labels))
	})

	t.Run("Error, 2 dynakube point to same namespace", func(t *testing.T) {
		dk2 := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				MonitoredNamespaces: &metav1.LabelSelector{MatchLabels: labels},
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		}
		clt := fake.NewClient(dk, dk2)

		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()
		assert.Error(t, err)
		assert.False(t, updated)
	})

	t.Run("Remove stale namespace entry", func(t *testing.T) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					InstanceLabel: dk.Name,
				},
			},
		}
		clt := fake.NewClient(dk)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()
		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, 0, len(nm.targetNs.Labels))
	})
	t.Run("Allow multiple dynakubes with different features", func(t *testing.T) {
		labels := map[string]string{"test": "selector"}
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
				Name:   "test-namespace",
				Labels: labels,
			},
		}
		clt := fake.NewClient(differentDk1, differentDk2)
		nm := NewNamespaceMapper(context.TODO(), clt, clt, "dynatrace", namespace, logger.NewDTLogger())

		updated, err := nm.MapFromNamespace()
		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, 2, len(nm.targetNs.Labels))
	})
}
