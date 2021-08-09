package oneagent

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildLabels(t *testing.T) {
	l := buildLabels("my-name", "classic")
	assert.Equal(t, l["dynatrace.com/component"], "operator")
	assert.Equal(t, l["operator.dynatrace.com/instance"], "my-name")
	assert.Equal(t, l["operator.dynatrace.com/feature"], "classic")
}

func newOneAgent() *dynatracev1alpha1.DynaKube {
	return &dynatracev1alpha1.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DynaKube",
			APIVersion: "dynatrace.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-oneagent",
			Namespace: "my-namespace",
			UID:       "69e98f18-805a-42de-84b5-3eae66534f75",
		},
	}
}
