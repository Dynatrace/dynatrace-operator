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

func NewRegular(instance *edgeconnectv1alpha1.EdgeConnect) *appsv1.Deployment {
	return create(instance, instance.Spec.OAuth.ClientSecret, instance.Spec.OAuth.Resource)
}

func NewProvisioner(instance *edgeconnectv1alpha1.EdgeConnect, clientSecretName string, resource string) *appsv1.Deployment {
	return create(instance, clientSecretName, resource)
}

func create(instance *edgeconnectv1alpha1.EdgeConnect, clientSecretName string, resource string) *appsv1.Deployment {
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
					Containers:                    []corev1.Container{edgeConnectContainer(instance, resource)},
					ImagePullSecrets:              prepareImagePullSecrets(instance),
					ServiceAccountName:            consts.EdgeConnectServiceAccountName,
					TerminationGracePeriodSeconds: address.Of(int64(30)),
					Volumes:                       []corev1.Volume{prepareVolume(clientSecretName)},
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

func prepareContainerEnvVars(instance *edgeconnectv1alpha1.EdgeConnect, resource string) []corev1.EnvVar {
	envMap := prioritymap.New(prioritymap.WithPriority(defaultEnvPriority))
	prioritymap.Append(envMap, []corev1.EnvVar{
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
			Value: resource,
		},
	})

	// Since HostRestrictions is optional we should not pass empty env var
	// otherwise edge-connect will fail
	if instance.Spec.HostRestrictions != "" {
		prioritymap.Append(envMap, corev1.EnvVar{
			Name:  consts.EnvEdgeConnectRestrictHostsTo,
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

func edgeConnectContainer(instance *edgeconnectv1alpha1.EdgeConnect, resource string) corev1.Container {
	return corev1.Container{
		Name:            consts.EdgeConnectContainerName,
		Image:           instance.Status.Version.ImageID,
		ImagePullPolicy: corev1.PullAlways,
		Env:             prepareContainerEnvVars(instance, resource),
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
			{MountPath: consts.EdgeConnectMountPath, Name: consts.EdgeConnectVolumeMountName, ReadOnly: true},
		},
	}
}

func prepareVolume(clientSecretName string) corev1.Volume {
	return corev1.Volume{
		Name: consts.EdgeConnectVolumeMountName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: clientSecretName,
				Items: []corev1.KeyToPath{
					{Key: consts.KeyEdgeConnectOauthClientID, Path: consts.PathEdgeConnectOauthClientID},
					{Key: consts.KeyEdgeConnectOauthClientSecret, Path: consts.PathEdgeConnectOauthClientSecret},
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
