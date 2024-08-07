package utils

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildTopologySpreadConstraints(topologySpreadConstraints []corev1.TopologySpreadConstraint, appLabels *labels.AppLabels) []corev1.TopologySpreadConstraint {
	if len(topologySpreadConstraints) > 0 {
		return topologySpreadConstraints
	} else {
		return []corev1.TopologySpreadConstraint{
			{
				MaxSkew:           1,
				TopologyKey:       "topology.kubernetes.io/zone",
				WhenUnsatisfiable: "ScheduleAnyway",
				LabelSelector:     &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()},
			},
			{
				MaxSkew:           1,
				TopologyKey:       "kubernetes.io/hostname",
				WhenUnsatisfiable: "DoNotSchedule",
				LabelSelector:     &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()},
			},
		}
	}
}

func BuildUpdateStrategy() appsv1.StatefulSetUpdateStrategy {
	partition := int32(0)

	return appsv1.StatefulSetUpdateStrategy{
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
			Partition: &partition,
		},
		Type: appsv1.RollingUpdateStatefulSetStrategyType,
	}
}
