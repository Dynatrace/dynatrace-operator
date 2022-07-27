package dynakube

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (controller *DynakubeController) determineDynaKubePhase(dynakube *dynatracev1beta1.DynaKube) bool {
	if dynakube.NeedsActiveGate() {
		activeGatePods, err := controller.numberOfMissingActiveGatePods(dynakube)
		if err != nil {
			log.Error(err, "activegate statefulset could not be accessed", "dynakube", dynakube.Name)
			return updatePhaseIfChanged(dynakube, dynatracev1beta1.Error)
		}
		if activeGatePods > 0 {
			log.Info("activegate statefulset is still deploying", "dynakube", dynakube.Name)
			return updatePhaseIfChanged(dynakube, dynatracev1beta1.Deploying)
		}
		if activeGatePods < 0 {
			log.Info("activegate statefulset not yet available", "dynakube", dynakube.Name)
			return updatePhaseIfChanged(dynakube, dynatracev1beta1.Deploying)
		}
	}

	if dynakube.CloudNativeFullstackMode() || dynakube.ClassicFullStackMode() || dynakube.HostMonitoringMode() {
		oneAgentPods, err := controller.numberOfMissingOneagentPods(dynakube)
		if k8serrors.IsNotFound(err) {
			log.Info("oneagent daemonset not yet available", "dynakube", dynakube.Name)
			return updatePhaseIfChanged(dynakube, dynatracev1beta1.Deploying)
		}
		if err != nil {
			log.Error(err, "oneagent daemonset could not be accessed", "dynakube", dynakube.Name)
			return updatePhaseIfChanged(dynakube, dynatracev1beta1.Error)
		}
		if oneAgentPods > 0 {
			log.Info("oneagent daemonset is still deploying", "dynakube", dynakube.Name)
			return updatePhaseIfChanged(dynakube, dynatracev1beta1.Deploying)
		}
	}

	return updatePhaseIfChanged(dynakube, dynatracev1beta1.Running)
}

func (controller *DynakubeController) numberOfMissingOneagentPods(dynakube *dynatracev1beta1.DynaKube) (int32, error) {
	oneAgentDaemonSet := &appsv1.DaemonSet{}
	instanceName := dynakube.OneAgentDaemonsetName()
	err := controller.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: dynakube.Namespace}, oneAgentDaemonSet)

	if err != nil {
		return 0, err
	}
	return oneAgentDaemonSet.Status.CurrentNumberScheduled - oneAgentDaemonSet.Status.NumberReady, nil
}

func (controller *DynakubeController) numberOfMissingActiveGatePods(dynakube *dynatracev1beta1.DynaKube) (int32, error) {
	capabilities := generateActiveGateCapabilities(dynakube)

	sum := int32(0)
	capabilityFound := false

	for _, activeGateCapability := range capabilities {
		activeGateStatefulSet := &appsv1.StatefulSet{}
		instanceName := capability.CalculateStatefulSetName(activeGateCapability, dynakube.Name)
		err := controller.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: dynakube.Namespace}, activeGateStatefulSet)

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
