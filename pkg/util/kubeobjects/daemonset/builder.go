package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/builder"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Mandatory fields, provided in constructor as named params
	setName      = builder.SetName[*appsv1.DaemonSet]
	setNamespace = builder.SetNamespace[*appsv1.DaemonSet]

	// Optional fields, provided in constructor as list of options
	SetLabels = builder.SetLabels[*appsv1.DaemonSet]
)

func Build(owner metav1.Object, name string, container corev1.Container, options ...builder.Option[*appsv1.DaemonSet]) (*appsv1.DaemonSet, error) {
	neededOpts := []builder.Option[*appsv1.DaemonSet]{
		setName(name),
		SetContainer(container),
		setNamespace(owner.GetNamespace()),
	}
	neededOpts = append(neededOpts, options...)

	return builder.Build(owner, &appsv1.DaemonSet{}, neededOpts...)
}

func SetAffinity(affinity corev1.Affinity) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.Spec.Template.Spec.Affinity = &affinity
	}
}

func SetVolumes(volumes []corev1.Volume) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.Spec.Template.Spec.Volumes = volumes
	}
}

func SetTolerations(tolerations []corev1.Toleration) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.Spec.Template.Spec.Tolerations = tolerations
	}
}

func SetDNSPolicy(policy corev1.DNSPolicy) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.Spec.Template.Spec.DNSPolicy = policy
	}
}

func SetPriorityClass(className string) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.Spec.Template.Spec.PriorityClassName = className
	}
}

func SetPullSecret(secretRef ...corev1.LocalObjectReference) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.Spec.Template.Spec.ImagePullSecrets = append(s.Spec.Template.Spec.ImagePullSecrets, secretRef...)
	}
}

func SetAllLabels(labels, matchLabels, templateLabels, customLabels map[string]string) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.ObjectMeta.Labels = labels
		s.Spec.Selector = &metav1.LabelSelector{MatchLabels: matchLabels}
		s.Spec.Template.ObjectMeta.Labels = maputils.MergeMap(customLabels, templateLabels)
	}
}

func SetAllAnnotations(annotations, templateAnnotations map[string]string) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.ObjectMeta.Annotations = maputils.MergeMap(s.ObjectMeta.Annotations, annotations)
		s.Spec.Template.ObjectMeta.Annotations = maputils.MergeMap(s.Spec.Template.ObjectMeta.Annotations, templateAnnotations)
	}
}

func SetContainer(container corev1.Container) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
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

func SetInitContainer(initContainer corev1.Container) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		targetIndex := 0
		for index := range s.Spec.Template.Spec.InitContainers {
			if s.Spec.Template.Spec.InitContainers[targetIndex].Name == initContainer.Name {
				targetIndex = index

				break
			}
		}

		if targetIndex == 0 {
			s.Spec.Template.Spec.InitContainers = make([]corev1.Container, 1)
		}

		s.Spec.Template.Spec.InitContainers[targetIndex] = initContainer
	}
}

func SetServiceAccount(serviceAccountName string) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		s.Spec.Template.Spec.DeprecatedServiceAccount = serviceAccountName
	}
}

func SetUpdateStrategy(updateStartegy appsv1.DaemonSetUpdateStrategy) builder.Option[*appsv1.DaemonSet] {
	return func(s *appsv1.DaemonSet) {
		s.Spec.UpdateStrategy = updateStartegy
	}
}
