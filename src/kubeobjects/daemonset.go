package kubeobjects

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdateDaemonSet(kubernetesClient client.Client, logger logr.Logger, desiredDaemonSet *appsv1.DaemonSet) (bool, error) {
	currentDaemonSet, err := getDaemonSet(kubernetesClient, desiredDaemonSet)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		logger.Info("creating new daemonset", "name", desiredDaemonSet.Name)
		return true, kubernetesClient.Create(context.TODO(), desiredDaemonSet)
	} else if err != nil {
		return false, err
	}

	if !HashAnnotationChanged(currentDaemonSet, desiredDaemonSet) {
		return false, nil
	}

	if LabelsNotEqual(currentDaemonSet.Spec.Selector.MatchLabels, desiredDaemonSet.Spec.Selector.MatchLabels) {
		return recreateDaemonSet(kubernetesClient, logger, currentDaemonSet, desiredDaemonSet)
	}

	logger.Info("updating existing daemonset", "name", desiredDaemonSet.Name)
	if err = kubernetesClient.Update(context.TODO(), desiredDaemonSet); err != nil {
		return false, err
	}
	return true, err
}

func recreateDaemonSet(kubernetesClient client.Client, logger logr.Logger, currentDs, desiredDaemonSet *appsv1.DaemonSet) (bool, error) {
	logger.Info("immutable section changed on daemonset, deleting and recreating", "name", desiredDaemonSet.Name)
	err := kubernetesClient.Delete(context.TODO(), currentDs)
	if err != nil {
		return false, err
	}
	logger.Info("deleted daemonset")
	logger.Info("recreating daemonset", "name", desiredDaemonSet.Name)
	return true, kubernetesClient.Create(context.TODO(), desiredDaemonSet)
}

func getDaemonSet(kubernetesClient client.Client, desiredDaemonSet *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	var actualDaemonSet appsv1.DaemonSet
	err := kubernetesClient.Get(
		context.TODO(),
		client.ObjectKey{
			Name:      desiredDaemonSet.Name,
			Namespace: desiredDaemonSet.Namespace,
		},
		&actualDaemonSet,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &actualDaemonSet, nil
}
