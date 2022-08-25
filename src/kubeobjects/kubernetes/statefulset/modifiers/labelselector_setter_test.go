package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/types"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLabelSelectorSetter(t *testing.T) {
	t.Run("Set labelselector", func(t *testing.T) {

		labelSelector := &v1.LabelSelector{
			MatchLabels: types.Labels{"a": "aa"},
		}

		b := statefulset.Builder{}
		b.AddModifier(
			LabelSelectorSetter{LabelSelector: labelSelector},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Selector: labelSelector,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
