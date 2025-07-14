package deployment

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/resources"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	unprivilegedUser  = int64(1000)
	unprivilegedGroup = int64(1000)
)

func New(ec *edgeconnect.EdgeConnect) *appsv1.Deployment {
	return create(ec)
}

func create(ec *edgeconnect.EdgeConnect) *appsv1.Deployment {
	appLabels := buildAppLabels(ec)
	labels := appLabels.BuildLabels()

	customPodLabels := maputils.MergeMap(
		ec.Spec.Labels,
		labels, // higher priority
	)

	log.Debug("EdgeConnect deployment app labels", "labels", labels)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ec.Name,
			Namespace:   ec.Namespace,
			Labels:      labels,
			Annotations: buildAnnotations(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ec.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: appLabels.BuildMatchLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: ec.Spec.Annotations,
					Labels:      customPodLabels,
				},
				Spec: corev1.PodSpec{
					Containers:                    []corev1.Container{edgeConnectContainer(ec)},
					ImagePullSecrets:              prepareImagePullSecrets(ec),
					ServiceAccountName:            ec.GetServiceAccountName(),
					DeprecatedServiceAccount:      ec.GetServiceAccountName(),
					TerminationGracePeriodSeconds: ptr.To(int64(30)),
					Volumes:                       prepareVolumes(ec),
					NodeSelector:                  ec.Spec.NodeSelector,
					Tolerations:                   ec.Spec.Tolerations,
					TopologySpreadConstraints:     ec.Spec.TopologySpreadConstraints,
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

func prepareImagePullSecrets(ec *edgeconnect.EdgeConnect) []corev1.LocalObjectReference {
	if ec.Spec.CustomPullSecret != "" {
		return []corev1.LocalObjectReference{
			{Name: ec.Spec.CustomPullSecret},
		}
	}

	return nil
}

func buildAppLabels(ec *edgeconnect.EdgeConnect) *labels.AppLabels {
	return labels.NewAppLabels(
		labels.EdgeConnectComponentLabel,
		ec.Name,
		consts.EdgeConnectUserProvisioned,
		ec.Status.Version.Version)
}

func buildAnnotations() map[string]string {
	return map[string]string{
		consts.AnnotationEdgeConnectContainerAppArmor: "runtime/default",
		webhook.AnnotationDynatraceInject:             "false",
	}
}

func edgeConnectContainer(ec *edgeconnect.EdgeConnect) corev1.Container {
	return corev1.Container{
		Name:            consts.EdgeConnectContainerName,
		Image:           ec.Status.Version.ImageID,
		ImagePullPolicy: corev1.PullAlways,
		Env:             ec.Spec.Env,
		Resources:       prepareResourceRequirements(ec),
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Privileged:               ptr.To(false),
			ReadOnlyRootFilesystem:   ptr.To(true),
			RunAsGroup:               ptr.To(unprivilegedGroup),
			RunAsUser:                ptr.To(unprivilegedUser),
			RunAsNonRoot:             ptr.To(true),
		},
		VolumeMounts: prepareVolumeMounts(ec),
	}
}

func prepareVolumes(ec *edgeconnect.EdgeConnect) []corev1.Volume {
	volumes := []corev1.Volume{prepareConfigVolume(ec)}

	if ec.Spec.CaCertsRef != "" {
		volumes = append(volumes, corev1.Volume{
			Name: consts.EdgeConnectCustomCAVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ec.Spec.CaCertsRef,
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

func prepareVolumeMounts(ec *edgeconnect.EdgeConnect) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{MountPath: consts.EdgeConnectConfigPath, SubPath: consts.EdgeConnectConfigFileName, Name: ec.Name + "-" + consts.EdgeConnectConfigVolumeMountName},
	}

	if ec.Spec.CaCertsRef != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{MountPath: consts.EdgeConnectMountPath, Name: consts.EdgeConnectCustomCAVolumeName})
	}

	return volumeMounts
}

func prepareConfigVolume(ec *edgeconnect.EdgeConnect) corev1.Volume {
	return corev1.Volume{
		Name: ec.Name + "-" + consts.EdgeConnectConfigVolumeMountName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: ec.Name + "-" + consts.EdgeConnectSecretSuffix,
				Items: []corev1.KeyToPath{
					{Key: consts.EdgeConnectConfigFileName, Path: consts.EdgeConnectConfigFileName},
				},
			},
		},
	}
}

func prepareResourceRequirements(ec *edgeconnect.EdgeConnect) corev1.ResourceRequirements {
	limits := resources.NewResourceList("100m", "128Mi")
	requests := resources.NewResourceList("100m", "128Mi")

	if ec.Spec.Resources.Limits != nil {
		limits = ec.Spec.Resources.Limits
	}

	if ec.Spec.Resources.Requests != nil {
		requests = ec.Spec.Resources.Requests
	}

	return corev1.ResourceRequirements{
		Requests: requests,
		Limits:   limits,
	}
}
