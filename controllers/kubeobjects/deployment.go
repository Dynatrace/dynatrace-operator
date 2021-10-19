package kubeobjects

import (
	"context"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
