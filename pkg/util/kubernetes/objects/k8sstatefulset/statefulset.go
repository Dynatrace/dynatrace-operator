package k8sstatefulset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ResolveAndSetReplicas(ctx context.Context, c client.Client, r client.Reader, log logd.Logger, ss *appsv1.StatefulSet, defaultReplicas *int32) error {
	replicas, err := ResolveReplicas(ctx, c, r, log, ss.Name, ss.Namespace, defaultReplicas)

	if err != nil {
		return err
	}

	ss.Spec.Replicas = ptr.To(replicas)
	return nil
}

func ResolveReplicas(ctx context.Context, c client.Client, r client.Reader, log logd.Logger, ssName, ssNamespace string, defaultReplicas *int32) (int32, error) {
	if defaultReplicas != nil {
		return *defaultReplicas, nil
	}

	obj, err := Query(c, r, log).Get(ctx, client.ObjectKey{Namespace: ssNamespace, Name: ssName})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return 1, nil
		}
		return 0, err
	}
	return GetReplicas(obj), nil
}

func GetReplicas(ss *appsv1.StatefulSet) int32 {
	switch {
	case ss == nil:
		return 0
	case ss.Spec.Replicas == nil:
		return 1
	default:
		return *ss.Spec.Replicas
	}
}
