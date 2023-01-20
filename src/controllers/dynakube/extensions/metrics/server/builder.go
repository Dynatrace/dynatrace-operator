package server

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/extensions/metrics/common"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	replicas                    = 1
	revisionHistoryLimit        = 10
	progressDeadlineSeconds     = 600
	rollingUpdateMaxUnavailable = "25%"
	rollingUpdateMaxSurge       = "25%"

	envBaseUrl = "BASE_URL"

	serviceAccount = "dynatrace-metric-server"

	dynaMetricMountName = "dynametric"
	dynaMetricMountPath = "/var/lib/dynatrace/secrets/dynametric"
	tmpStorageMountName = "tmp-storage"
	tmpStorageMountPath = "/tmp"
)

var (
	tmpStorageMountSizeLimit = kubeobjects.NewQuantity("1Mi")

	requestResources = kubeobjects.NewResources("50m", "64Mi")
	limitResources   = kubeobjects.NewResources("100m", "128Mi")
)

type builder struct {
	*dynatracev1beta1.DynaKube
	*kubeobjects.AppLabels
}

func newBuilder(dynaKube *dynatracev1beta1.DynaKube) *builder {
	return &builder{
		DynaKube: dynaKube,
		AppLabels: kubeobjects.NewAppLabels(
			common.KubjectNamePrefix,
			dynaKube.Name,
			kubeobjects.ExtApiComponentLabel,
			kubeobjects.CustomImageLabelValue,
		),
	}
}

func (builder *builder) newDeployment() (*appsv1.Deployment, error) {
	objMeta := metav1.ObjectMeta{
		Name:      common.KubjectNamePrefix,
		Namespace: builder.DynaKube.Namespace,
		Labels:    builder.AppLabels.BuildLabels(),
	}

	deployment := appsv1.Deployment{
		ObjectMeta: objMeta,
		Spec: appsv1.DeploymentSpec{
			Replicas:                address.Of[int32](replicas),
			RevisionHistoryLimit:    address.Of[int32](revisionHistoryLimit),
			ProgressDeadlineSeconds: address.Of[int32](progressDeadlineSeconds),

			Selector: &metav1.LabelSelector{
				MatchLabels: builder.AppLabels.BuildMatchLabels(),
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: address.Of(intstr.FromString(rollingUpdateMaxUnavailable)),
					MaxSurge:       address.Of(intstr.FromString(rollingUpdateMaxSurge)),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: objMeta,
				Spec: corev1.PodSpec{
					Volumes: builder.volumes(),
					Containers: []corev1.Container{
						builder.container(),
					},

					ServiceAccountName: serviceAccount,
					SecurityContext:    builder.podSecurityContext(),
				},
			},
		},
	}

	hash, err := kubeobjects.GenerateHash(deployment)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	deployment.ObjectMeta.Annotations = map[string]string{
		kubeobjects.AnnotationHash: hash,
	}

	return &deployment, nil
}

func (builder *builder) volumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: dynaMetricMountName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: builder.DynaKube.Spec.Synthetic.DynaMetrics.Token,
				},
			},
		},
		{
			Name: tmpStorageMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: tmpStorageMountSizeLimit,
				},
			},
		},
	}
}

func (builder *builder) container() corev1.Container {
	return corev1.Container{
		Name:            common.KubjectNamePrefix,
		Image:           builder.image(),
		ImagePullPolicy: corev1.PullAlways,
		Env:             builder.envVars(),
		Args:            builder.args(),
		Ports:           builder.ports(),
		VolumeMounts:    builder.volumeMounts(),
		SecurityContext: builder.securityContext(),
		Resources:       builder.resources(),
	}
}

func (builder *builder) image() string {
	return builder.DynaKube.DynaMetricImage()
}

func (builder *builder) envVars() []corev1.EnvVar {
	vars := []corev1.EnvVar{
		{
			Name: envBaseUrl,
			Value: fmt.Sprintf(
				"https://%s/e/%s",
				builder.DynaKube.ApiUrlHost(),
				builder.DynaKube.Status.ConnectionInfo.TenantUUID),
		},
	}

	return append(
		vars,
		builder.DynaKube.Spec.Synthetic.DynaMetrics.Env...)
}

func (*builder) args() []string {
	return []string{
		fmt.Sprintf("--secure-port=%d", common.HttpsServicePort),
		fmt.Sprintf("--cert-dir=%s", tmpStorageMountPath),
	}
}

func (*builder) volumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      dynaMetricMountName,
			MountPath: dynaMetricMountPath,
		},
		{
			Name:      tmpStorageMountName,
			MountPath: tmpStorageMountPath,
		},
	}
}

func (*builder) ports() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          common.HttpsServicePortName,
			ContainerPort: common.HttpsServicePort,
		},
	}
}

func (*builder) securityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               address.Of(false),
		AllowPrivilegeEscalation: address.Of(false),
		ReadOnlyRootFilesystem:   address.Of(true),
		RunAsNonRoot:             address.Of(true),

		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
	}
}

func (*builder) podSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsUser:           address.Of[int64](1001),
		FSGroup:             address.Of[int64](1001),
		FSGroupChangePolicy: address.Of(corev1.FSGroupChangeOnRootMismatch),
	}
}

func (*builder) resources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits:   limitResources,
		Requests: requestResources,
	}
}
