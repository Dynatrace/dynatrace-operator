package topology

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MaxOnePerNode(appLabels *labels.AppLabels) []corev1.TopologySpreadConstraint {
	nodeInclusionPolicyHonor := corev1.NodeInclusionPolicyHonor

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
			NodeTaintsPolicy:  &nodeInclusionPolicyHonor,
			LabelSelector:     &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()},
		},
	}
}
