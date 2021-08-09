package kubeobjects

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdateDaemonSet(c client.Client, logger logr.Logger, desiredDs *appsv1.DaemonSet) (bool, error) {
	currentDs, err := getDaemonSet(c, desiredDs)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		logger.Info("creating new daemonset for CSI driver")
		return true, c.Create(context.TODO(), desiredDs)
	} else if err != nil {
		return false, nil
	}

	if !HasChanged(currentDs, desiredDs) {
		return false, nil
	}

	logger.Info("updating existing daemonset for CSI driver")
	if err = c.Update(context.TODO(), desiredDs); err != nil {
		return false, err
	}
	return true, err
}

func getDaemonSet(c client.Client, desiredDs *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	var actualDs appsv1.DaemonSet
	err := c.Get(context.TODO(), client.ObjectKey{Name: desiredDs.Name, Namespace: desiredDs.Namespace}, &actualDs)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &actualDs, nil
}
