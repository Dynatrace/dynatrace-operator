package deployment

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/builder"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var SetLabels = builder.SetLabels[*appsv1.Deployment]

func Build(owner metav1.Object, name string, options ...builder.Option[*appsv1.Deployment]) (*appsv1.Deployment, error) {
	neededOpts := []builder.Option[*appsv1.Deployment]{
		builder.SetName[*appsv1.Deployment](name),
		builder.SetNamespace[*appsv1.Deployment](owner.GetNamespace()),
	}
	neededOpts = append(neededOpts, options...)

	return builder.Build(owner, &appsv1.Deployment{}, neededOpts...)
}

func SetReplicas(replicas int32) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		d.Spec.Replicas = ptr.To(replicas)
	}
}

func SetAffinity(afinity *corev1.Affinity) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Affinity = afinity
	}
}

func SetTolerations(tolerations []corev1.Toleration) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Tolerations = tolerations
	}
}

func SetTopologySpreadConstraints(topologySpreadConstraints []corev1.TopologySpreadConstraint) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.TopologySpreadConstraints = topologySpreadConstraints
	}
}

func SetAllLabels(labels, matchLabels, templateLabels map[string]string) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		d.Labels = labels
		d.Spec.Selector = &metav1.LabelSelector{MatchLabels: matchLabels}
		d.Spec.Template.Labels = templateLabels
	}
}

func SetAllAnnotations(annotations, templateAnnotations map[string]string) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		d.Annotations = maputils.MergeMap(d.Annotations, annotations)
		d.Spec.Template.Annotations = maputils.MergeMap(d.Spec.Template.Annotations, templateAnnotations)
	}
}

func SetContainer(container corev1.Container) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		targetIndex := 0
		for index := range d.Spec.Template.Spec.Containers {
			if d.Spec.Template.Spec.Containers[targetIndex].Name == container.Name {
				targetIndex = index

				break
			}
		}

		if targetIndex == 0 {
			d.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
		}

		d.Spec.Template.Spec.Containers[targetIndex] = container
	}
}

func SetServiceAccount(serviceAccountName string) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.ServiceAccountName = serviceAccountName
		d.Spec.Template.Spec.DeprecatedServiceAccount = serviceAccountName
	}
}

func SetSecurityContext(securityContext *corev1.PodSecurityContext) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.SecurityContext = securityContext
	}
}

func SetNodeSelector(nodeSelector map[string]string) builder.Option[*appsv1.Deployment] {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.NodeSelector = nodeSelector
	}
}

func SetImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) func(o *appsv1.Deployment) {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}
}

func SetVolumes(volumes []corev1.Volume) func(o *appsv1.Deployment) {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Volumes = volumes
	}
}
