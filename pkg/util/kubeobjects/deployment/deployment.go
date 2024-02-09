package deployment

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/object"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployment returns the Deployment object who is the owner of this pod.
func GetDeployment(c client.Client, podName, namespace string) (*appsv1.Deployment, error) {
	var pod corev1.Pod

	err := c.Get(context.TODO(), client.ObjectKey{Name: podName, Namespace: namespace}, &pod)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rsOwner := metav1.GetControllerOf(&pod)
	if rsOwner == nil {
		return nil, errors.Errorf("no controller found for Pod: %s", pod.Name)
	} else if rsOwner.Kind != "ReplicaSet" {
		return nil, errors.Errorf("unexpected controller found for Pod: %s, kind: %s", pod.Name, rsOwner.Kind)
	}

	var rs appsv1.ReplicaSet
	if err := c.Get(context.TODO(), client.ObjectKey{Name: rsOwner.Name, Namespace: namespace}, &rs); err != nil {
		return nil, errors.WithStack(err)
	}

	dOwner := metav1.GetControllerOf(&rs)
	if dOwner == nil {
		return nil, errors.Errorf("no controller found for ReplicaSet: %s", rs.Name)
	} else if dOwner.Kind != "Deployment" {
		return nil, errors.Errorf("unexpected controller found for ReplicaSet: %s, kind: %s", pod.Name, dOwner.Kind)
	}

	var d appsv1.Deployment
	if err := c.Get(context.TODO(), client.ObjectKey{Name: dOwner.Name, Namespace: namespace}, &d); err != nil {
		return nil, errors.WithStack(err)
	}

	return &d, nil
}

func CreateOrUpdateDeployment(c client.Client, logger logger.DtLogger, desiredDeployment *appsv1.Deployment) (bool, error) {
	currentDeployment, err := getDeployment(c, desiredDeployment)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		logger.Info("creating new deployment", "name", desiredDeployment.Name)

		return true, c.Create(context.TODO(), desiredDeployment)
	} else if err != nil {
		return false, err
	}

	if !hasher.IsAnnotationDifferent(currentDeployment, desiredDeployment) {
		return false, nil
	}

	if labels.NotEqual(currentDeployment.Spec.Selector.MatchLabels, desiredDeployment.Spec.Selector.MatchLabels) {
		logger.Info("immutable section changed on deployment, deleting and recreating", "name", desiredDeployment.Name)

		return recreateDeployment(c, logger, currentDeployment, desiredDeployment)
	}

	logger.Info("updating existing deployment", "name", desiredDeployment.Name)

	if err = c.Update(context.TODO(), desiredDeployment); err != nil {
		return false, err
	}

	return true, err
}

func getDeployment(c client.Client, desiredDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	var actualDaemonSet appsv1.Deployment

	err := c.Get(
		context.TODO(),
		object.Key(desiredDeployment),
		&actualDaemonSet,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &actualDaemonSet, nil
}

func recreateDeployment(c client.Client, logger logger.DtLogger, currentDs, desiredDeployment *appsv1.Deployment) (bool, error) {
	err := c.Delete(context.TODO(), currentDs)
	if err != nil {
		return false, err
	}

	logger.Info("deleted deployment")
	logger.Info("recreating deployment", "name", desiredDeployment.Name)

	return true, c.Create(context.TODO(), desiredDeployment)
}
