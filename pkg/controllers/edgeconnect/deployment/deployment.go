package deployment

import (
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/resources"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	customEnvPriority  = prioritymap.HighPriority
	defaultEnvPriority = prioritymap.DefaultPriority
	unprivilegedUser   = int64(1000)
	unprivilegedGroup  = int64(1000)
)

func New(instance *edgeconnectv1alpha1.EdgeConnect) *appsv1.Deployment {
	return create(instance)
}

func create(instance *edgeconnectv1alpha1.EdgeConnect) *appsv1.Deployment {
	appLabels := buildAppLabels(instance)
	labels := maputils.MergeMap(
		instance.Labels,
		appLabels.BuildLabels(),
	)

	log.Debug("EdgeConnect deployment app labels", "labels", labels)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: buildAnnotations(instance),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: appLabels.BuildMatchLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers:                    []corev1.Container{edgeConnectContainer(instance)},
					ImagePullSecrets:              prepareImagePullSecrets(instance),
					ServiceAccountName:            instance.Spec.ServiceAccountName,
					DeprecatedServiceAccount:      instance.Spec.ServiceAccountName,
					TerminationGracePeriodSeconds: address.Of(int64(30)),
					Volumes:                       prepareVolumes(instance),
					NodeSelector:                  instance.Spec.NodeSelector,
					Tolerations:                   instance.Spec.Tolerations,
					TopologySpreadConstraints:     instance.Spec.TopologySpreadConstraints,
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				// default is already 25%
				RollingUpdate: &appsv1.RollingUpdateDeployment{},
			},
			MinReadySeconds:         0,
			RevisionHistoryLimit:    nil,
			Paused:                  false,
			ProgressDeadlineSeconds: nil,
		},
	}
}

func prepareImagePullSecrets(instance *edgeconnectv1alpha1.EdgeConnect) []corev1.LocalObjectReference {
	if instance.Spec.CustomPullSecret != "" {
		return []corev1.LocalObjectReference{
			{Name: instance.Spec.CustomPullSecret},
		}
	}

	return nil
}

func buildAppLabels(instance *edgeconnectv1alpha1.EdgeConnect) *labels.AppLabels {
	return labels.NewAppLabels(
		labels.EdgeConnectComponentLabel,
		instance.Name,
		consts.EdgeConnectUserProvisioned,
		instance.Status.Version.Version)
}

func buildAnnotations(instance *edgeconnectv1alpha1.EdgeConnect) map[string]string {
	annotations := map[string]string{
		consts.AnnotationEdgeConnectContainerAppArmor: "runtime/default",
		webhook.AnnotationDynatraceInject:             "false",
	}
	annotations = maputils.MergeMap(instance.Annotations, annotations)

	return annotations
}

func edgeConnectContainer(instance *edgeconnectv1alpha1.EdgeConnect) corev1.Container {
	return corev1.Container{
		Name:            consts.EdgeConnectContainerName,
		Image:           instance.Status.Version.ImageID,
		ImagePullPolicy: corev1.PullAlways,
		Env:             instance.Spec.Env,
		Resources:       prepareResourceRequirements(instance),
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: address.Of(false),
			Privileged:               address.Of(false),
			ReadOnlyRootFilesystem:   address.Of(true),
			RunAsGroup:               address.Of(unprivilegedGroup),
			RunAsUser:                address.Of(unprivilegedUser),
			RunAsNonRoot:             address.Of(true),
		},
		VolumeMounts: prepareVolumeMounts(instance),
	}
}

func prepareVolumes(instance *edgeconnectv1alpha1.EdgeConnect) []corev1.Volume {
	volumes := []corev1.Volume{prepareConfigVolume(instance)}

	if instance.Spec.CaCertsRef != "" {
		volumes = append(volumes, corev1.Volume{
			Name: consts.EdgeConnectCustomCAVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.Spec.CaCertsRef,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  consts.EdgeConnectCAConfigMapKey,
							Path: consts.EdgeConnectCustomCertificateName,
						},
					},
				},
			},
		})
	}

	return volumes
}

func prepareVolumeMounts(instance *edgeconnectv1alpha1.EdgeConnect) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{MountPath: consts.EdgeConnectConfigPath, SubPath: consts.EdgeConnectConfigFileName, Name: instance.Name + "-" + consts.EdgeConnectConfigVolumeMountName},
	}

	if instance.Spec.CaCertsRef != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{MountPath: consts.EdgeConnectMountPath, Name: consts.EdgeConnectCustomCAVolumeName})
	}

	return volumeMounts
}

func prepareConfigVolume(instance *edgeconnectv1alpha1.EdgeConnect) corev1.Volume {
	return corev1.Volume{
		Name: instance.Name + "-" + consts.EdgeConnectConfigVolumeMountName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: instance.Name + "-" + consts.EdgeConnectSecretSuffix,
				Items: []corev1.KeyToPath{
					{Key: consts.EdgeConnectConfigFileName, Path: consts.EdgeConnectConfigFileName},
				},
			},
		},
	}
}

func prepareResourceRequirements(instance *edgeconnectv1alpha1.EdgeConnect) corev1.ResourceRequirements {
	limits := resources.NewResourceList("100m", "128Mi")
	requests := resources.NewResourceList("100m", "128Mi")

	if instance.Spec.Resources.Limits != nil {
		limits = instance.Spec.Resources.Limits
	}

	if instance.Spec.Resources.Requests != nil {
		requests = instance.Spec.Resources.Requests
	}

	return corev1.ResourceRequirements{
		Requests: requests,
		Limits:   limits,
	}
}
