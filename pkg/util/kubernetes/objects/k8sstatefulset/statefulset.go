// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sstatefulset

import (
	"context"
	"errors"

	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrRolloutInProgress = errors.New("statefulset rollout in progress")

func IsRolloutComplete(statefulSet *appsv1.StatefulSet) bool {
	if statefulSet == nil {
		return false
	}

	desiredReplicas := ptr.Deref(statefulSet.Spec.Replicas, int32(0))

	return statefulSet.Generation == statefulSet.Status.ObservedGeneration && desiredReplicas == statefulSet.Status.ReadyReplicas
}

func ResolveAndSetReplicas(ctx context.Context, r client.Reader, ss *appsv1.StatefulSet, defaultReplicas *int32) error {
	replicas, err := ResolveReplicas(ctx, r, client.ObjectKeyFromObject(ss), defaultReplicas)
	if err != nil {
		return err
	}

	ss.Spec.Replicas = new(replicas)

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

	return ptr.Deref(ss.Spec.Replicas, 1), nil
}
