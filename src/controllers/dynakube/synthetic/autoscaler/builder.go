package autoscaler

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	appsv1 "k8s.io/api/apps/v1"
	scalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type builder struct {
	*dynatracev1beta1.DynaKube
	*appsv1.StatefulSet
	*kubeobjects.AppLabels
}

var (
	externalMetricTargetValueMetricQuantity = kubeobjects.NewQuantity("80")

	behaviorScaleUpStabilizationWindowSeconds   = address.Of[int32](0)
	behaviorScaleDownStabilizationWindowSeconds = address.Of[int32](300)

	hpaScalingPolicies = []scalingv2.HPAScalingPolicy{
		{
			Type:          scalingv2.PodsScalingPolicy,
			Value:         1,
			PeriodSeconds: 600,
		},
	}
)

func newBuilder(
	dynakube *dynatracev1beta1.DynaKube,
	statefulSet *appsv1.StatefulSet,
) *builder {
	return &builder{
		DynaKube:    dynakube,
		StatefulSet: statefulSet,
		AppLabels: kubeobjects.NewAppLabels(
			SynAutoscaler,
			dynakube.Name,
			kubeobjects.SyntheticComponentLabel,
			kubeobjects.CustomImageLabelValue,
		),
	}
}

func (builder *builder) newAutoscaler() *scalingv2.HorizontalPodAutoscaler {
	return &scalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.DynaKube.Name + "-" + SynAutoscaler,
			Namespace: builder.DynaKube.Namespace,
			Labels:    builder.AppLabels.BuildLabels(),
		},
		Spec: scalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: scalingv2.CrossVersionObjectReference{
				Kind: builder.StatefulSet.GetObjectKind().
					GroupVersionKind().
					Kind,
				Name:       builder.StatefulSet.GetName(),
				APIVersion: builder.StatefulSet.TypeMeta.APIVersion,
			},
			MinReplicas: address.Of(builder.DynaKube.FeatureSyntheticAutoscalerMinReplicas()),
			MaxReplicas: builder.DynaKube.FeatureSyntheticAutoscalerMaxReplicas(),
			Metrics: []scalingv2.MetricSpec{
				{
					Type: scalingv2.ExternalMetricSourceType,
					External: &scalingv2.ExternalMetricSource{
						Metric: scalingv2.MetricIdentifier{
							Name: builder.DynaKube.FeatureSyntheticAutoscalerDynaQuery(),
						},
						Target: scalingv2.MetricTarget{
							Type:  scalingv2.ValueMetricType,
							Value: externalMetricTargetValueMetricQuantity,
						},
					},
				},
			},
			Behavior: &scalingv2.HorizontalPodAutoscalerBehavior{
				ScaleUp: &scalingv2.HPAScalingRules{
					StabilizationWindowSeconds: behaviorScaleUpStabilizationWindowSeconds,
					SelectPolicy:               address.Of(scalingv2.MinChangePolicySelect),
					Policies:                   hpaScalingPolicies,
				},
				ScaleDown: &scalingv2.HPAScalingRules{
					StabilizationWindowSeconds: behaviorScaleDownStabilizationWindowSeconds,
					SelectPolicy:               address.Of(scalingv2.MinChangePolicySelect),
					Policies:                   hpaScalingPolicies,
				},
			},
		},
	}
}
