package dynakube

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (controller *DynakubeController) determineDynaKubePhase(instance *dynatracev1beta1.DynaKube) bool {
	if instance.NeedsActiveGate() {
		activeGatePods, err := controller.numberOfMissingActiveGatePods(instance)
		if err != nil {
			log.Error(err, "activegate sts could not be accessed", "dynakube", instance.Name)
			return updatePhaseIfChanged(instance, dynatracev1beta1.Error)
		}
		if activeGatePods > 0 {
			log.Info("activegate sts is still deploying", "dynakube", instance.Name)
			return updatePhaseIfChanged(instance, dynatracev1beta1.Deploying)
		}
		if activeGatePods == 0 {
			log.Info("activegate sts not yet available", "dynakube", instance.Name)
			return updatePhaseIfChanged(instance, dynatracev1beta1.Deploying)
		}
	}

	if instance.CloudNativeFullstackMode() || instance.ClassicFullStackMode() || instance.HostMonitoringMode() {
		oneAgentPods, err := controller.numberOfMissingOneagentPods(instance)
		if k8serrors.IsNotFound(err) {
			log.Info("oneagent daemonset not yet available", "dynakube", instance.Name)
			return updatePhaseIfChanged(instance, dynatracev1beta1.Deploying)
		}
		if err != nil {
			log.Error(err, "oneagent daemonset could not be accessed", "dynakube", instance.Name)
			return updatePhaseIfChanged(instance, dynatracev1beta1.Error)
		}
		if oneAgentPods > 0 {
			log.Info("oneagent daemonset is still deploying", "dynakube", instance.Name)
			return updatePhaseIfChanged(instance, dynatracev1beta1.Deploying)
		}
	}

	return updatePhaseIfChanged(instance, dynatracev1beta1.Running)
}

func (controller *DynakubeController) numberOfMissingOneagentPods(instance *dynatracev1beta1.DynaKube) (int32, error) {
	dsActual := &appsv1.DaemonSet{}
	instanceName := instance.OneAgentDaemonsetName()
	err := controller.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: instance.Namespace}, dsActual)

	if err != nil {
		return 0, err
	}
	return dsActual.Status.CurrentNumberScheduled - dsActual.Status.NumberReady, nil
}

func (controller *DynakubeController) numberOfMissingActiveGatePods(instance *dynatracev1beta1.DynaKube) (int32, error) {
	capabilities := generateActiveGateCapabilities(instance)

	sum := int32(0)
	capabilityFound := false

	for _, c := range capabilities {
		stsActual := &appsv1.StatefulSet{}
		instanceName := capability.CalculateStatefulSetName(c, instance.Name)
		err := controller.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: instance.Namespace}, stsActual)

		if k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return -1, err
		}
		capabilityFound = true
		sum += stsActual.Status.CurrentReplicas - stsActual.Status.ReadyReplicas
	}

	if !capabilityFound {
		return 0, nil
	}

	return sum, nil
}
