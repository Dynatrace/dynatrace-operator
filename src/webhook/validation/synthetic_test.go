package validation

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyntheticInvalidSettings(t *testing.T) {
	const (
		invalidType = "XL"
		replicas    = "3"
	)
	dynaKube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId:      "unknown",
				dynatracev1beta1.AnnotationFeatureSyntheticNodeType:              invalidType,
				dynatracev1beta1.AnnotationFeatureSyntheticAutoscalerMinReplicas: replicas,
				dynatracev1beta1.AnnotationFeatureSyntheticAutoscalerMaxReplicas: replicas,
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

	toAssertInvalidReplicaBounds := func(t *testing.T) {
		assertDeniedResponse(
			t,
			[]string{errorInvalidSyntheticAutoscalerReplicaBounds},
			&dynaKube)
	}
	t.Run("autoscaler-replica-bounds", toAssertInvalidReplicaBounds)

	toAssertUndefinedDynaMetricsToken := func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorInvalidDynaMetricsToken},
			&dynaKube)
	}
	t.Run("dynametrics-token", toAssertUndefinedDynaMetricsToken)
}
