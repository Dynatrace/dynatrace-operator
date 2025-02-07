package job

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/builder"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Mandatory fields, provided in constructor as named params
	setName      = builder.SetName[*batchv1.Job]
	setNamespace = builder.SetNamespace[*batchv1.Job]

	// Optional fields, provided in constructor as list of options
	SetLabels = builder.SetLabels[*batchv1.Job]
)

func Build(owner metav1.Object, name string, container corev1.Container, options ...builder.Option[*batchv1.Job]) (*batchv1.Job, error) {
	neededOpts := []builder.Option[*batchv1.Job]{
		setName(name),
		SetContainer(container),
		setNamespace(owner.GetNamespace()),
	}
	neededOpts = append(neededOpts, options...)

	return builder.Build(owner, &batchv1.Job{}, neededOpts...)
}

func SetVolumes(volumes []corev1.Volume) builder.Option[*batchv1.Job] {
	return func(s *batchv1.Job) {
		s.Spec.Template.Spec.Volumes = volumes
	}
}

func SetTolerations(tolerations []corev1.Toleration) builder.Option[*batchv1.Job] {
	return func(s *batchv1.Job) {
		s.Spec.Template.Spec.Tolerations = tolerations
	}
}

func SetNodeName(nodeName string) builder.Option[*batchv1.Job] {
	return func(s *batchv1.Job) {
		s.Spec.Template.Spec.NodeName = nodeName
	}
}

func SetOnFailureRestartPolicy() builder.Option[*batchv1.Job] {
	return func(s *batchv1.Job) {
		s.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
	}
}

func SetPullSecret(pullSecrets ...string) builder.Option[*batchv1.Job] {
	return func(s *batchv1.Job) {
		imagePullSecrets := make([]corev1.LocalObjectReference, 0)
		for _, pullSecretName := range pullSecrets {
			imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
				Name: pullSecretName,
			})
		}

		s.Spec.Template.Spec.ImagePullSecrets = append(s.Spec.Template.Spec.ImagePullSecrets, imagePullSecrets...)
	}
}

func SetAutomountServiceAccountToken(isEnabled bool) builder.Option[*batchv1.Job] {
	return func(s *batchv1.Job) {
		s.Spec.Template.Spec.AutomountServiceAccountToken = &isEnabled
	}
}

func SetPodAnnotations(annotations map[string]string) builder.Option[*batchv1.Job] {
	return func(s *batchv1.Job) {
		s.Spec.Template.ObjectMeta.Annotations = annotations
	}
}

func SetAllLabels(labels, matchLabels, templateLabels, customLabels map[string]string) builder.Option[*batchv1.Job] {
	return func(s *batchv1.Job) {
		s.ObjectMeta.Labels = labels
		if matchLabels == nil {
			s.Spec.Selector = &metav1.LabelSelector{MatchLabels: matchLabels}
		}

		s.Spec.Template.ObjectMeta.Labels = maputils.MergeMap(customLabels, templateLabels)
	}
}

func SetContainer(container corev1.Container) builder.Option[*batchv1.Job] {
	return func(s *batchv1.Job) {
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
