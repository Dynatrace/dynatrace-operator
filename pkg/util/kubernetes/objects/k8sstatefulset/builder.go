package k8sstatefulset

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/builder"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var (
	// Mandatory fields, provided in constructor as named params
	setName      = builder.SetName[*appsv1.StatefulSet]
	setNamespace = builder.SetNamespace[*appsv1.StatefulSet]

	// Optional fields, provided in constructor as list of options
	SetLabels = builder.SetLabels[*appsv1.StatefulSet]
)

func Build(owner metav1.Object, name string, container corev1.Container, options ...builder.Option[*appsv1.StatefulSet]) (*appsv1.StatefulSet, error) {
	neededOpts := slices.Concat([]builder.Option[*appsv1.StatefulSet]{
		setName(name),
		SetContainer(container),
		setNamespace(owner.GetNamespace()),
	}, options, []builder.Option[*appsv1.StatefulSet]{SetPVCAnnotation()})

	return builder.Build(owner, &appsv1.StatefulSet{}, neededOpts...)
}

func SetReplicas(replicas int32) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Spec.Replicas = ptr.To(replicas)
	}
}

func SetPodManagementPolicy(podManagementType appsv1.PodManagementPolicyType) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Spec.PodManagementPolicy = podManagementType
	}
}

func SetAffinity(afinity corev1.Affinity) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Spec.Template.Spec.Affinity = &afinity
	}
}

func SetTolerations(tolerations []corev1.Toleration) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Spec.Template.Spec.Tolerations = tolerations
	}
}

func SetTopologySpreadConstraints(topologySpreadConstraints []corev1.TopologySpreadConstraint) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Spec.Template.Spec.TopologySpreadConstraints = topologySpreadConstraints
	}
}

func SetAllLabels(labels, matchLabels, templateLabels, customLabels map[string]string) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Labels = labels
		s.Spec.Selector = &metav1.LabelSelector{MatchLabels: matchLabels}
		s.Spec.Template.Labels = maputils.MergeMap(customLabels, templateLabels)
	}
}

func SetAllAnnotations(annotations, templateAnnotations map[string]string) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Annotations = maputils.MergeMap(s.Annotations, annotations)
		s.Spec.Template.Annotations = maputils.MergeMap(s.Spec.Template.Annotations, templateAnnotations)
	}
}

func SetContainer(container corev1.Container) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		targetIndex := 0
		for index := range s.Spec.Template.Spec.Containers {
			if s.Spec.Template.Spec.Containers[targetIndex].Name == container.Name {
				targetIndex = index

				break
			}
		}

		if targetIndex == 0 {
			s.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
		}

		s.Spec.Template.Spec.Containers[targetIndex] = container
	}
}

func SetServiceAccount(serviceAccountName string) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		s.Spec.Template.Spec.DeprecatedServiceAccount = serviceAccountName
	}
}

func SetSecurityContext(securityContext *corev1.PodSecurityContext) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Spec.Template.Spec.SecurityContext = securityContext
	}
}

func SetRollingUpdateStrategyType() builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		s.Spec.UpdateStrategy = appsv1.StatefulSetUpdateStrategy{
			RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
				Partition: ptr.To(int32(0)),
			},
			Type: appsv1.RollingUpdateStatefulSetStrategyType,
		}
	}
}

func SetPVCAnnotation() builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		if s.Spec.VolumeClaimTemplates != nil {
			if s.Annotations == nil {
				s.Annotations = map[string]string{}
			}

			s.Annotations[AnnotationPVCHash], _ = hasher.GenerateHash(s.Spec.VolumeClaimTemplates)
		}
	}
}
