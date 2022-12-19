package validation

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyntheticInvalidSettings(t *testing.T) {
	capability := []dynatracev1beta1.CapabilityDisplayName{
		dynatracev1beta1.SyntheticCapability.DisplayName,
	}
	invalidType := "XL"

	t.Run("synthetic-node-type", func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{fmt.Sprintf(errorInvalidSyntheticNodeType, invalidType)},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
					Annotations: map[string]string{
						dynatracev1beta1.AnnotationFeatureSyntheticNodeType: invalidType,
					},
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: capability,
					},
				},
			})
	})
}
