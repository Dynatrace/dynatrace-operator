package codemodules

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMatchForNamespace(t *testing.T) {
	dynakubes := []dynatracev1alpha1.DynaKube{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
					Selector: metav1.LabelSelector{
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
					Selector: metav1.LabelSelector{
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
					Selector: metav1.LabelSelector{
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
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		}

		dynakube, err := matchForNamespace(dynakubes, namespace)
		assert.NoError(t, err)
		assert.Nil(t, dynakube)
	})

	t.Run(`Match namespace with labels`, func(t *testing.T) {
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					"inject": "true",
				},
			},
		}

		dynakube, err := matchForNamespace(dynakubes, namespace)
		assert.NoError(t, err)
		assert.NotNil(t, dynakube)
	})
	t.Run(`Match namespace with expressions`, func(t *testing.T) {
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					"type": "app",
				},
			},
		}

		dynakube, err := matchForNamespace(dynakubes, namespace)
		assert.NoError(t, err)
		assert.NotNil(t, dynakube)
	})
	t.Run(`Error on multiple Dynakube matches`, func(t *testing.T) {
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					"type":   "app",
					"inject": "true",
				},
			},
		}

		dynakube, err := matchForNamespace(dynakubes, namespace)
		assert.Error(t, err)
		assert.Nil(t, dynakube)
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

	codeModules, err := findCodeModules(context.TODO(), clt)
	assert.NoError(t, err)
	assert.NotNil(t, codeModules)
	assert.Equal(t, 2, len(codeModules))
}
