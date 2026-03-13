package k8sdeployment

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployment returns the Deployment object who is the owner of this pod.
// not doable using generics
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

func ResolveAndSetReplicas(ctx context.Context, r client.Reader, log logd.Logger, d *appsv1.Deployment, defaultReplicas *int32) error {
	replicas, err := ResolveReplicas(ctx, r, client.ObjectKey{Name: d.Name, Namespace: d.Namespace}, log, defaultReplicas)
	if err != nil {
		return err
	}

	d.Spec.Replicas = ptr.To(replicas)

	return nil
}

func ResolveReplicas(ctx context.Context, r client.Reader, key client.ObjectKey, log logd.Logger, defaultReplicas *int32) (int32, error) {
	if defaultReplicas != nil {
		return *defaultReplicas, nil
	}

	obj, err := Query(nil, r, log).Get(ctx, key)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return 1, nil
		}

		return 0, err
	}

	return GetReplicas(obj), nil
}

func GetReplicas(d *appsv1.Deployment) int32 {
	switch {
	case d == nil:
		return 0
	case d.Spec.Replicas == nil:
		return 1
	default:
		return *d.Spec.Replicas
	}
}
