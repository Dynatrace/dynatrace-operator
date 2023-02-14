package autoscaler

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	syntheticLocationEntityId    = "doctored"
	syntheticImage               = "nowhere/synthetic:archaic"
	syntheticAutoscalerDynaQuery = "inoperable.com?loc_id=%s"
)

var (
	syntheticAutoscalerMaxReplicas = int32(11)

	dynaKube = &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ephemeral",
			Namespace: "experimental",
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId:      syntheticLocationEntityId,
				dynatracev1beta1.AnnotationFeatureCustomSyntheticImage:           syntheticImage,
				dynatracev1beta1.AnnotationFeatureSyntheticAutoscalerMaxReplicas: fmt.Sprint(syntheticAutoscalerMaxReplicas),
				dynatracev1beta1.AnnotationFeatureSyntheticAutoscalerDynaQuery:   syntheticAutoscalerDynaQuery,
			},
		},
	}

	statefulSet = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transient",
			Namespace: "experimental",
			Labels: map[string]string{
				"ignored": "undefined",
			},
		},
	}
)

func TestSyntheticAutoscaler(t *testing.T) {
	assertion := assert.New(t)
	autoscaler, err := newBuilder(dynaKube, statefulSet).newAutoscaler()

	toAssertScaleTargetReference := func(t *testing.T) {
		require.NoError(t, err)
		assertion.Equal(
			autoscaler.Spec.ScaleTargetRef.Name,
			statefulSet.Name,
			"declared scale target name: %s",
			statefulSet.Name)
	}
	t.Run("by-scale-target-reference", toAssertScaleTargetReference)

	toAssertLabels := func(t *testing.T) {
		assertion.Equal(
			autoscaler.ObjectMeta.Labels,
			statefulSet.ObjectMeta.Labels,
			"declared labels: %v",
			statefulSet.ObjectMeta.Labels)
	}
	t.Run("by-labels", toAssertLabels)

	toAssertMinReplicas := func(t *testing.T) {
		assertion.Equal(
			*autoscaler.Spec.MinReplicas,
			dynatracev1beta1.DefaultSyntheticAutoscalerMinReplicas,
			"declared min replicas: %s",
			dynatracev1beta1.DefaultSyntheticAutoscalerMinReplicas)
	}
	t.Run("by-min-replicas", toAssertMinReplicas)

	toAssertMaxReplicas := func(t *testing.T) {
		assertion.Equal(
			autoscaler.Spec.MaxReplicas,
			syntheticAutoscalerMaxReplicas,
			"declared max replicas: %s",
			syntheticAutoscalerMaxReplicas)
	}
	t.Run("by-max-replicas", toAssertMaxReplicas)

	resolved := fmt.Sprintf(syntheticAutoscalerDynaQuery, syntheticLocationEntityId)
	toAssertDynaQuery := func(t *testing.T) {
		assertion.Equal(
			autoscaler.Spec.Metrics[0].External.Metric.Name,
			resolved,
			"declared ext metric query: %s",
			resolved)
	}
	t.Run("by-ext-metric-query", toAssertDynaQuery)
}
