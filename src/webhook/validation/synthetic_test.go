package validation

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyntheticInvalidSettings(t *testing.T) {
	const (
		invalidType     = "XL"
		invalidReplicas = "?"
	)
	dynaKube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId: "unknown",
				dynatracev1beta1.AnnotationFeatureSyntheticNodeType:         invalidType,
				dynatracev1beta1.AnnotationFeatureSyntheticReplicas:         invalidReplicas,
			},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
		},
	}

	toAssertInvalidNodeType := func(t *testing.T) {
		assertDeniedResponse(
			t,
			[]string{fmt.Sprintf(errorInvalidSyntheticNodeType, invalidType)},
			&dynaKube)
	}
	t.Run("node-type", toAssertInvalidNodeType)

	delete(
		dynaKube.ObjectMeta.Annotations,
		dynatracev1beta1.AnnotationFeatureSyntheticNodeType)
	toAssertDefaultReplicas := func(t *testing.T) {
		assertAllowedResponseWithWarnings(t, 2, &dynaKube)
	}
	t.Run("valid-replicas", toAssertDefaultReplicas)
}
