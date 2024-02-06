package dynakube

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
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

	t.Run("node type", func(t *testing.T) {
		assertDeniedResponse(
			t,
			[]string{fmt.Sprintf(errorInvalidSyntheticNodeType, invalidType)},
			&dynaKube)
	})

	delete(
		dynaKube.ObjectMeta.Annotations,
		dynatracev1beta1.AnnotationFeatureSyntheticNodeType)
	t.Run("valid replicas", func(t *testing.T) {
		assertAllowedResponseWithWarnings(t, 2, &dynaKube)
	})
}
