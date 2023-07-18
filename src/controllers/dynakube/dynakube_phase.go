package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (controller *Controller) determineDynaKubePhase(dynakube *dynatracev1beta1.DynaKube) status.DeploymentPhase { // nolint:revive
	if dynakube.NeedsActiveGate() {
		activeGatePods, err := controller.numberOfMissingActiveGatePods(dynakube)
		if err != nil {
			log.Error(err, "activegate statefulset could not be accessed", "dynakube", dynakube.Name)
			return status.Error
		}
		if activeGatePods > 0 {
			log.Info("activegate statefulset is still deploying", "dynakube", dynakube.Name)
			return status.Deploying
		}
		if activeGatePods < 0 {
			log.Info("activegate statefulset not yet available", "dynakube", dynakube.Name)
			return status.Deploying
		}
	}

	if dynakube.CloudNativeFullstackMode() || dynakube.ClassicFullStackMode() || dynakube.HostMonitoringMode() {
		oneAgentPods, err := controller.numberOfMissingOneagentPods(dynakube)
		if k8serrors.IsNotFound(err) {
			log.Info("oneagent daemonset not yet available", "dynakube", dynakube.Name)
			return status.Deploying
		}
		if err != nil {
			log.Error(err, "oneagent daemonset could not be accessed", "dynakube", dynakube.Name)
			return status.Error
		}
		if oneAgentPods > 0 {
			log.Info("oneagent daemonset is still deploying", "dynakube", dynakube.Name)
			return status.Deploying
		}
	}

	if dynakube.Status.NamespaceSecretsPhase == status.Error {
		log.Info("secrets not ready because of an error", "dynakube", dynakube.Name)
		return status.Error
	}

	if !controller.secretsReady(dynakube) {
		log.Info("secrets not ready", "dynakube", dynakube.Name)
		return status.Deploying
	}

	return status.Running
}

func (controller *Controller) numberOfMissingOneagentPods(dynakube *dynatracev1beta1.DynaKube) (int32, error) {
	oneAgentDaemonSet := &appsv1.DaemonSet{}
	instanceName := dynakube.OneAgentDaemonsetName()
	err := controller.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: dynakube.Namespace}, oneAgentDaemonSet)

	if err != nil {
		return 0, err
	}
	return oneAgentDaemonSet.Status.CurrentNumberScheduled - oneAgentDaemonSet.Status.NumberReady, nil
}

func (controller *Controller) numberOfMissingActiveGatePods(dynakube *dynatracev1beta1.DynaKube) (int32, error) {
	capabilities := capability.GenerateActiveGateCapabilities(dynakube)

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

func (controller *Controller) secretsReady(dynakube *dynatracev1beta1.DynaKube) bool {
	/*
		dynakubeNamespaceList, err := mapper.GetNamespacesForDynakube(context.TODO(), controller.apiReader, dynakube.Name)
		if err != nil {
			log.Info("namespace list error", "dynakubeName", dynakube.Name, "err", err)
			return false
		}
	*/
	dynakubeNamespaceList, err := controller.countFromDynakube(dynakube)
	if err != nil {
		log.Info("namespace list error", "dynakubeName", dynakube.Name, "err", err)
		return false
	}

	for _, namespace := range dynakubeNamespaceList {
		log.Info("namespace list", "name", namespace.Name)
	}

	for _, secretName := range []string{config.EnrichmentEndpointSecretName, config.AgentInitSecretName} {
		secretList, err := mapper.GetSecrets(context.TODO(), controller.apiReader, secretName)
		if err != nil {
			log.Info("secret list error", "err", err)
			return false
		}

		for _, secret := range secretList {
			log.Info("secret list", "name", secret.Name, "namespace", secret.Namespace)
		}

		for _, namespace := range dynakubeNamespaceList {
			if !slices.ContainsFunc(secretList, func(secret corev1.Secret) bool {
				return secret.Namespace == namespace.Name
			}) {
				log.Info("secret not found", "secretName", secretName, "namespaceName", namespace.Name)
				return false
			}
		}
	}
	return true
}

func (controller *Controller) countFromDynakube(dynakube *dynatracev1beta1.DynaKube) ([]*corev1.Namespace, error) {
	if !dynakube.NeedAppInjection() {
		log.Info("dynakube doesn't need AppInjection", "namespace", dynakube.Namespace, "dynakube", dynakube.Name)
		return []*corev1.Namespace{}, nil
	}

	nsList := &corev1.NamespaceList{}
	if err := controller.apiReader.List(context.TODO(), nsList); err != nil {
		return []*corev1.Namespace{}, errors.Cause(err)
	}

	var matchingNamespaces []*corev1.Namespace
	for i := range nsList.Items {
		matched, err := mapper.IsMatchingNamespace(nsList.Items[i].Name, nsList.Items[i].Labels, dynakube)
		if err != nil {
			return []*corev1.Namespace{}, err
		}
		if matched {
			matchingNamespaces = append(matchingNamespaces, &nsList.Items[i])
		}
	}
	return matchingNamespaces, nil
}
