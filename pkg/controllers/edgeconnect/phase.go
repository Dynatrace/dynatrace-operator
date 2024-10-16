package edgeconnect

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (controller *Controller) determineEdgeConnectPhase(ec *edgeconnect.EdgeConnect) status.DeploymentPhase {
	deployment := &appsv1.Deployment{}

	err := controller.client.Get(context.Background(), types.NamespacedName{Name: ec.Name, Namespace: ec.Namespace}, deployment)
	if k8serrors.IsNotFound(err) {
		log.Info("edgeConnect deployment to be deployed", "edgeConnect", ec.Name, "deployment", ec.Name)

		return status.Deploying
	}

	if err != nil {
		log.Error(err, "edgeConnect deployment could not be accessed", "edgeConnect", ec.Name, "namespace", ec.Namespace)

		return status.Error
	}

	scheduledReplicas := int32(0)
	if deployment.Spec.Replicas != nil {
		scheduledReplicas = *deployment.Spec.Replicas
	}

	if scheduledReplicas != deployment.Status.ReadyReplicas {
		log.Info("edgeConnect deployment is still deploying", "edgeConnect", ec.Name, "namespace", ec.Name)

		return status.Deploying
	}

	return status.Running
}
