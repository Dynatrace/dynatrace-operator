package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (controller *Controller) determineDynaKubePhase(dk *dynakube.DynaKube) status.DeploymentPhase {
	if dk.NeedsActiveGate() {
		activeGatePods, err := controller.numberOfMissingActiveGatePods(dk)
		if err != nil {
			log.Error(err, "activegate statefulset could not be accessed", "dynakube", dk.Name)

			return status.Error
		}

		if activeGatePods > 0 {
			log.Info("activegate statefulset is still deploying", "dynakube", dk.Name)

			return status.Deploying
		}

		if activeGatePods < 0 {
			log.Info("activegate statefulset not yet available", "dynakube", dk.Name)

			return status.Deploying
		}
	}

	if dk.CloudNativeFullstackMode() || dk.ClassicFullStackMode() || dk.HostMonitoringMode() {
		oneAgentPods, err := controller.numberOfMissingOneagentPods(dk)
		if k8serrors.IsNotFound(err) {
			log.Info("oneagent daemonset not yet available", "dynakube", dk.Name)

			return status.Deploying
		}

		if err != nil {
			log.Error(err, "oneagent daemonset could not be accessed", "dynakube", dk.Name)

			return status.Error
		}

		if oneAgentPods > 0 {
			log.Info("oneagent daemonset is still deploying", "dynakube", dk.Name)

			return status.Deploying
		}
	}

	return status.Running
}

func (controller *Controller) numberOfMissingOneagentPods(dk *dynakube.DynaKube) (int32, error) {
	oneAgentDaemonSet := &appsv1.DaemonSet{}
	instanceName := dk.OneAgentDaemonsetName()

	err := controller.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: dk.Namespace}, oneAgentDaemonSet)
	if err != nil {
		return 0, err
	}

	return oneAgentDaemonSet.Status.CurrentNumberScheduled - oneAgentDaemonSet.Status.NumberReady, nil
}

func (controller *Controller) numberOfMissingActiveGatePods(dk *dynakube.DynaKube) (int32, error) {
	capabilities := capability.GenerateActiveGateCapabilities(dk)

	sum := int32(0)
	capabilityFound := false

	for _, activeGateCapability := range capabilities {
		activeGateStatefulSet := &appsv1.StatefulSet{}
		instanceName := capability.CalculateStatefulSetName(activeGateCapability, dk.Name)

		err := controller.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: dk.Namespace}, activeGateStatefulSet)
		if k8serrors.IsNotFound(err) {
			continue
		}

		if err != nil {
			return -1, err
		}

		capabilityFound = true

		// This check is needed as in our unit tests replicas is always nil. We can't set it manually as this function
		// is called from the same function where the statefulset is created
		scheduledReplicas := int32(0)
		if activeGateStatefulSet.Spec.Replicas != nil {
			scheduledReplicas = *activeGateStatefulSet.Spec.Replicas
		}

		sum += scheduledReplicas - activeGateStatefulSet.Status.ReadyReplicas
	}

	if !capabilityFound {
		return -1, nil
	}

	return sum, nil
}
