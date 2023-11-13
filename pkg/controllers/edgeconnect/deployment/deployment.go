package deployment

import (
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	edgeconnectconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/resources"
	utilmap "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
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
	appLabels := buildAppLabels(instance)
	labels := utilmap.MergeMap(
		instance.Labels,
		appLabels.BuildLabels(),
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
					ServiceAccountName:            edgeconnectconsts.EdgeConnectServiceAccountName,
					TerminationGracePeriodSeconds: address.Of(int64(30)),
					Volumes:                       []corev1.Volume{prepareVolume(instance)},
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

func prepareContainerEnvVars(instance *edgeconnectv1alpha1.EdgeConnect) []corev1.EnvVar {
	envMap := prioritymap.New(prioritymap.WithPriority(defaultEnvPriority))
	prioritymap.Append(envMap, []corev1.EnvVar{
		{
			Name:  edgeconnectconsts.EnvEdgeConnectName,
			Value: instance.ObjectMeta.Name,
		},
		{
			Name:  edgeconnectconsts.EnvEdgeConnectApiEndpointHost,
			Value: instance.Spec.ApiServer,
		},

		{
			Name:  edgeconnectconsts.EnvEdgeConnectOauthEndpoint,
			Value: instance.Spec.OAuth.Endpoint,
		},
		{
			Name:  edgeconnectconsts.EnvEdgeConnectOauthResource,
			Value: instance.Spec.OAuth.Resource,
		},
	})

	// Since HostRestrictions is optional we should not pass empty env var
	// otherwise edge-connect will fail
	if instance.Spec.HostRestrictions != "" {
		prioritymap.Append(envMap, corev1.EnvVar{
			Name:  edgeconnectconsts.EnvEdgeConnectRestrictHostsTo,
			Value: instance.Spec.HostRestrictions,
		})
	}

	prioritymap.Append(envMap, instance.Spec.Env, prioritymap.WithPriority(customEnvPriority))

	return envMap.AsEnvVars()
}

func buildAppLabels(instance *edgeconnectv1alpha1.EdgeConnect) *labels.AppLabels {
	return labels.NewAppLabels(
		labels.EdgeConnectComponentLabel,
		instance.Name,
		edgeconnectconsts.EdgeConnectUserProvisioned,
		instance.Status.Version.Version)
}

func buildAnnotations(instance *edgeconnectv1alpha1.EdgeConnect) map[string]string {
	annotations := map[string]string{
		edgeconnectconsts.AnnotationEdgeConnectContainerAppArmor: "runtime/default",
		webhook.AnnotationDynatraceInject:                        "false",
	}
	annotations = utilmap.MergeMap(instance.Annotations, annotations)
	return annotations
}

func edgeConnectContainer(instance *edgeconnectv1alpha1.EdgeConnect) corev1.Container {
	return corev1.Container{
		Name:            edgeconnectconsts.EdgeConnectContainerName,
		Image:           instance.Status.Version.ImageID,
		ImagePullPolicy: corev1.PullAlways,
		Env:             prepareContainerEnvVars(instance),
		Resources:       prepareResourceRequirements(instance),
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: address.Of(false),
			Privileged:               address.Of(false),
			ReadOnlyRootFilesystem:   address.Of(true),
			RunAsGroup:               address.Of(unprivilegedGroup),
			RunAsUser:                address.Of(unprivilegedUser),
			RunAsNonRoot:             address.Of(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{MountPath: edgeconnectconsts.EdgeConnectMountPath, Name: edgeconnectconsts.EdgeConnectVolumeMountName, ReadOnly: true},
		},
	}
}

func prepareVolume(instance *edgeconnectv1alpha1.EdgeConnect) corev1.Volume {
	return corev1.Volume{
		Name: edgeconnectconsts.EdgeConnectVolumeMountName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: instance.Spec.OAuth.ClientSecret,
				Items: []corev1.KeyToPath{
					{Key: edgeconnectconsts.KeyEdgeConnectOauthClientID, Path: edgeconnectconsts.PathEdgeConnectOauthClientID},
					{Key: edgeconnectconsts.KeyEdgeConnectOauthClientSecret, Path: edgeconnectconsts.PathEdgeConnectOauthClientSecret},
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
