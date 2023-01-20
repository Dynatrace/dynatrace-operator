package validation

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyntheticInvalidSettings(t *testing.T) {
	invalidType := "XL"
	dynaKube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			Synthetic: dynatracev1beta1.SyntheticSpec{
				NodeType: invalidType,
				Autoscaler: dynatracev1beta1.AutoscalerSpec{
					MinReplicas: 3,
					MaxReplicas: 3,
				},
			},
		},
	}

	toAssertInvalidNodeType := func(t *testing.T) {
		assertDeniedResponse(
			t,
			[]string{fmt.Sprintf(errorInvalidSyntheticNodeType, invalidType)},
			&dynaKube)
	}
	t.Run("node-type", toAssertInvalidNodeType)

	toAssertInvalidReplicaBounds := func(t *testing.T) {
		assertDeniedResponse(
			t,
			[]string{errorInvalidSyntheticAutoscalerReplicaBounds},
			&dynaKube)
	}
	t.Run("autoscaler-replica-bounds", toAssertInvalidReplicaBounds)
}
