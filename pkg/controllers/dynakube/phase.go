package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (controller *Controller) determineDynaKubePhase(dk *dynakube.DynaKube) status.DeploymentPhase {
	components := []func(dk *dynakube.DynaKube) status.DeploymentPhase{
		controller.determineActiveGatePhase,
		controller.determineExtensionsExecutionControllerPhase,
		controller.determineExtensionsCollectorPhase,
		controller.determineOneAgentPhase,
		controller.determineLogAgentPhase,
		controller.determineKSPMPhase,
	}
	for _, component := range components {
		if phase := component(dk); phase != status.Running {
			return phase
		}
	}

	return status.Running
}

func (controller *Controller) determineActiveGatePhase(dk *dynakube.DynaKube) status.DeploymentPhase {
	if dk.ActiveGate().IsEnabled() {
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

	return status.Running
}

func (controller *Controller) determineExtensionsExecutionControllerPhase(dk *dynakube.DynaKube) status.DeploymentPhase {
	return controller.determinePrometheusStatefulsetPhase(dk, dk.Extensions().GetExecutionControllerStatefulsetName())
}

func (controller *Controller) determineExtensionsCollectorPhase(dk *dynakube.DynaKube) status.DeploymentPhase {
	return controller.determinePrometheusStatefulsetPhase(dk, dk.OtelCollectorStatefulsetName())
}

func (controller *Controller) determinePrometheusStatefulsetPhase(dk *dynakube.DynaKube, statefulsetName string) status.DeploymentPhase {
	if dk.Extensions().IsPrometheusEnabled() {
		statefulSet := &appsv1.StatefulSet{}

		err := controller.client.Get(context.Background(), types.NamespacedName{Name: statefulsetName, Namespace: dk.Namespace}, statefulSet)
		if k8serrors.IsNotFound(err) {
			log.Info("statefulset to be deployed", "dynakube", dk.Name, "statefulset", statefulsetName)

			return status.Deploying
		}

		if err != nil {
			log.Error(err, "statefulset could not be accessed", "dynakube", dk.Name, "statefulset", statefulsetName)

			return status.Error
		}

		scheduledReplicas := int32(0)
		if statefulSet.Spec.Replicas != nil {
			scheduledReplicas = *statefulSet.Spec.Replicas
		}

		if scheduledReplicas != statefulSet.Status.ReadyReplicas {
			log.Info("statefulset is still deploying", "dynakube", dk.Name, "statefulset", statefulsetName)

			return status.Deploying
		}
	}

	return status.Running
}

func (controller *Controller) determineOneAgentPhase(dk *dynakube.DynaKube) status.DeploymentPhase {
	if dk.OneAgent().IsCloudNativeFullstackMode() || dk.OneAgent().IsClassicFullStackMode() || dk.OneAgent().IsHostMonitoringMode() {
		oneAgentPods, err := controller.numberOfMissingDaemonSetPods(dk, dk.OneAgent().GetDaemonsetName())
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

func (controller *Controller) determineLogAgentPhase(dk *dynakube.DynaKube) status.DeploymentPhase {
	if dk.LogMonitoring().IsStandalone() {
		logAgentPods, err := controller.numberOfMissingDaemonSetPods(dk, dk.LogMonitoring().GetDaemonSetName())
		if k8serrors.IsNotFound(err) {
			log.Info("logagent daemonset not yet available", "dynakube", dk.Name)

			return status.Deploying
		}

		if err != nil {
			log.Error(err, "logagent daemonset could not be accessed", "dynakube", dk.Name)

			return status.Error
		}

		if logAgentPods > 0 {
			log.Info("logagent daemonset is still deploying", "dynakube", dk.Name)

			return status.Deploying
		}
	}

	return status.Running
}

func (controller *Controller) determineKSPMPhase(dk *dynakube.DynaKube) status.DeploymentPhase {
	if dk.KSPM().IsEnabled() {
		kspmPods, err := controller.numberOfMissingDaemonSetPods(dk, dk.KSPM().GetDaemonSetName())
		if k8serrors.IsNotFound(err) {
			log.Info("kspm daemonset not yet available", "dynakube", dk.Name)

			return status.Deploying
		}

		if err != nil {
			log.Error(err, "kspm daemonset could not be accessed", "dynakube", dk.Name)

			return status.Error
		}

		if kspmPods > 0 {
			log.Info("kspm daemonset is still deploying", "dynakube", dk.Name)

			return status.Deploying
		}
	}

	return status.Running
}

func (controller *Controller) numberOfMissingDaemonSetPods(dk *dynakube.DynaKube, dsName string) (int32, error) {
	daemonSet := &appsv1.DaemonSet{}
	instanceName := dsName

	err := controller.client.Get(context.Background(), types.NamespacedName{Name: instanceName, Namespace: dk.Namespace}, daemonSet)
	if err != nil {
		return 0, err
	}

	return daemonSet.Status.CurrentNumberScheduled - daemonSet.Status.NumberReady, nil
}

func (controller *Controller) numberOfMissingActiveGatePods(dk *dynakube.DynaKube) (int32, error) {
	activeGateStatefulSet := &appsv1.StatefulSet{}
	instanceName := capability.CalculateStatefulSetName(dk.Name)

	err := controller.client.Get(context.Background(), types.NamespacedName{Name: instanceName, Namespace: dk.Namespace}, activeGateStatefulSet)
	if k8serrors.IsNotFound(err) {
		return -1, nil
	}

	if err != nil {
		return -1, err
	}

	// This check is needed as in our unit tests replicas is always nil. We can't set it manually as this function
	// is called from the same function where the statefulset is created
	scheduledReplicas := int32(0)
	if activeGateStatefulSet.Spec.Replicas != nil {
		scheduledReplicas = *activeGateStatefulSet.Spec.Replicas
	}

	return scheduledReplicas - activeGateStatefulSet.Status.ReadyReplicas, nil
}
