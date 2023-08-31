package deployment

import (
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func New(instance *edgeconnectv1alpha1.EdgeConnect) *appsv1.Deployment {
	appLabels := buildAppLabels()
	labels := kubeobjects.MergeMap(
		instance.Labels,
		appLabels.BuildLabels(),
		map[string]string{kubeobjects.AppInstanceLabel: instance.ObjectMeta.Name},
	)

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
					Containers: []corev1.Container{edgeConnectContainer(instance)},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: instance.Spec.CustomPullSecret},
					},
					ServiceAccountName:            consts.EdgeConnectServiceAccountName,
					TerminationGracePeriodSeconds: address.Of(int64(30)),
					Volumes:                       []corev1.Volume{prepareVolume(instance)},
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

func prepareContainerEnvVars(instance *edgeconnectv1alpha1.EdgeConnect) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name:  consts.EnvEdgeConnectName,
			Value: instance.ObjectMeta.Name,
		},
		{
			Name:  consts.EnvEdgeConnectApiEndpointHost,
			Value: instance.Spec.ApiServer,
		},

		{
			Name:  consts.EnvEdgeConnectOauthEndpoint,
			Value: instance.Spec.OAuth.Endpoint,
		},
		{
			Name:  consts.EnvEdgeConnectOauthResource,
			Value: instance.Spec.OAuth.Resource,
		},
	}
	// Since HostRestrictions is optional we should not pass empty env var
	// otherwise edge-connect will fail
	if instance.Spec.HostRestrictions != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  consts.EnvEdgeConnectRestrictHostsTo,
			Value: instance.Spec.HostRestrictions,
		})
	}
	return envVars
}

func buildAppLabels() *kubeobjects.AppLabels {
	return kubeobjects.NewAppLabels(
		// appName =
		kubeobjects.EdgeConnectComponentLabel,
		// name =
		kubeobjects.EdgeConnectComponentLabel,
		// component =
		kubeobjects.EdgeConnectComponentLabel,
		// version =
		// NB: as of now edgeConnect doesn't have any version
		"latest")
}

func buildAnnotations(instance *edgeconnectv1alpha1.EdgeConnect) map[string]string {
	annotations := map[string]string{
		consts.AnnotationEdgeConnectContainerAppArmor: "runtime/default",
		webhook.AnnotationDynatraceInject:             "false",
	}
	annotations = kubeobjects.MergeMap(instance.Annotations, annotations)
	return annotations
}

func edgeConnectContainer(instance *edgeconnectv1alpha1.EdgeConnect) corev1.Container {
	return corev1.Container{
		Name:            consts.EdgeConnectContainerName,
		Image:           instance.Status.Version.ImageID,
		ImagePullPolicy: corev1.PullAlways,
		Env:             prepareContainerEnvVars(instance),
		Resources: corev1.ResourceRequirements{
			Requests: kubeobjects.NewResources("100m", "128Mi"),
			Limits:   kubeobjects.NewResources("100m", "128Mi"),
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: address.Of(false),
			Privileged:               address.Of(false),
			ReadOnlyRootFilesystem:   address.Of(true),
			RunAsGroup:               address.Of(kubeobjects.UnprivilegedGroup),
			RunAsUser:                address.Of(kubeobjects.UnprivilegedUser),
			RunAsNonRoot:             address.Of(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{MountPath: consts.EdgeConnectMountPath, Name: consts.EdgeConnectVolumeMountName, ReadOnly: true},
		},
	}
}

func prepareVolume(instance *edgeconnectv1alpha1.EdgeConnect) corev1.Volume {
	return corev1.Volume{
		Name: consts.EdgeConnectVolumeMountName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: instance.Spec.OAuth.ClientSecret,
				Items: []corev1.KeyToPath{
					{Key: "oauth-client-id", Path: "oauth/client_id"},
					{Key: "oauth-client-secret", Path: "oauth/client_secret"},
				},
			},
		},
	}
}
