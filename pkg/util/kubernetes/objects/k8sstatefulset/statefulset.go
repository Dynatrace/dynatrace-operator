package k8sstatefulset

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ResolveAndSetReplicas(ctx context.Context, r client.Reader, ss *appsv1.StatefulSet, defaultReplicas *int32) error {
	replicas, err := ResolveReplicas(ctx, r, client.ObjectKeyFromObject(ss), defaultReplicas)
	if err != nil {
		return err
	}

	ss.Spec.Replicas = ptr.To(replicas)

	return nil
}

func ResolveReplicas(ctx context.Context, r client.Reader, key client.ObjectKey, defaultReplicas *int32) (int32, error) {
	if defaultReplicas != nil {
		return *defaultReplicas, nil
	}

	ss := &appsv1.StatefulSet{}
	if err := r.Get(ctx, key, ss); err != nil {
		if k8serrors.IsNotFound(err) {
			return 1, nil
		}

		return 0, err
	}

	return getReplicas(ss), nil
}

func getReplicas(ss *appsv1.StatefulSet) int32 {
	switch {
	case ss == nil:
		return 0
	case ss.Spec.Replicas == nil:
		return 1
	default:
		return *ss.Spec.Replicas
	}
}
