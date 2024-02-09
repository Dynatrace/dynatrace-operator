package daemonset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/object"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdateDaemonSet(kubernetesClient client.Client, logger logger.DtLogger, desiredDaemonSet *appsv1.DaemonSet) (bool, error) {
	currentDaemonSet, err := getDaemonSet(kubernetesClient, desiredDaemonSet)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("creating new daemonset", "name", desiredDaemonSet.Name)

			return true, kubernetesClient.Create(context.TODO(), desiredDaemonSet)
		}

		return false, err
	}

	if !hasher.IsAnnotationDifferent(currentDaemonSet, desiredDaemonSet) {
		return false, nil
	}

	if labels.NotEqual(currentDaemonSet.Spec.Selector.MatchLabels, desiredDaemonSet.Spec.Selector.MatchLabels) {
		return recreateDaemonSet(kubernetesClient, logger, currentDaemonSet, desiredDaemonSet)
	}

	logger.Info("updating existing daemonset", "name", desiredDaemonSet.Name)

	if err = kubernetesClient.Update(context.TODO(), desiredDaemonSet); err != nil {
		return false, err
	}

	return true, err
}

func recreateDaemonSet(kubernetesClient client.Client, logger logger.DtLogger, currentDs, desiredDaemonSet *appsv1.DaemonSet) (bool, error) {
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
		object.Key(desiredDaemonSet),
		&actualDaemonSet,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &actualDaemonSet, nil
}
