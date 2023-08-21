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
	// prepare app labels
	appLabels := kubeobjects.NewAppLabels(
		kubeobjects.EdgeConnectComponentLabel,
		kubeobjects.EdgeConnectComponentLabel,
		kubeobjects.EdgeConnectComponentLabel,
		instance.Name)
	// build labels
	labels := kubeobjects.MergeMap(
		appLabels.BuildLabels(),
		map[string]string{kubeobjects.AppInstanceLabel: instance.ObjectMeta.Name},
	)

	// prepare annotations
	annotations := map[string]string{
		consts.AnnotationEdgeConnectContainerAppArmor: "runtime/default",
		webhook.AnnotationDynatraceInject:             "false",
	}

	replicas := int32(2)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: nil,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: nil,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
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
								{MountPath: "/etc/edge_connect", Name: "secrets", ReadOnly: true},
							},
						},
					},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: instance.Spec.CustomPullSecret},
					},
					ServiceAccountName:            "edgeconnect-dynatrace",
					TerminationGracePeriodSeconds: address.Of(int64(30)),
					Volumes: []corev1.Volume{
						{
							Name: "oauth-secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: instance.Spec.OAuth.ClientSecret,
									Items: []corev1.KeyToPath{
										{Key: "oauth-client-id", Path: "oauth/client_id"},
										{Key: "oauth-client-secret", Path: "oauth/client_secret"},
									},
								},
							},
						},
					},
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
	return []corev1.EnvVar{
		{
			Name:  "EDGE_CONNECT_NAME",
			Value: instance.ObjectMeta.Name,
		},
		{
			Name:  "EDGE_CONNECT_RESTRICT_HOSTS_TO",
			Value: instance.Spec.HostRestrictions,
		},
		{
			Name:  "EDGE_CONNECT_OAUTH__ENDPOINT",
			Value: instance.Spec.OAuth.Endpoint,
		},
		{
			Name:  "EDGE_CONNECT_OAUTH__RESOURCE",
			Value: instance.Spec.OAuth.Resource,
		},
	}
}
