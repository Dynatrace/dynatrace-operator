package deployment

import (
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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
		// TODO: <apparmour>
		webhook.AnnotationDynatraceInject: "false",
	}

	replicas := int32(2)
	//defaultRollingUpdate := "25%"
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: nil, // TOOD:
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.PodSpec{
					Volumes:        nil,
					InitContainers: nil,
					Containers: []corev1.Container{
						{
							Name:            "edge-connect",
							Image:           "test",
							ImagePullPolicy: "Always",
							Command:         nil,
							Args:            nil,
							EnvFrom:         nil,
							Env: []corev1.EnvVar{
								{
									Name:  "EDGE_CONNECT_NAME",
									Value: "",
								},
							},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    nil,
							LivenessProbe:   nil,
							ReadinessProbe:  nil,
							StartupProbe:    nil,
							Lifecycle:       nil,
							SecurityContext: nil,
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
