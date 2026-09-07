// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8ssecret

import (
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type QueryObject struct {
	query.Generic[*corev1.Secret, *corev1.SecretList]
}

func Query(kubeClient client.Client, kubeReader client.Reader) QueryObject {
	return QueryObject{
		query.Generic[*corev1.Secret, *corev1.SecretList]{
			Target:     &corev1.Secret{},
			ListTarget: &corev1.SecretList{},
			ToList: func(list *corev1.SecretList) []*corev1.Secret {
				out := make([]*corev1.Secret, len(list.Items))
				for i, item := range list.Items {
					out[i] = &item
				}

				return out
			},
			IsEqual:      isEqual,
			MustRecreate: mustRecreate,

			KubeClient: kubeClient,
			KubeReader: kubeReader,
		},
	}
}

func isEqual(secret *corev1.Secret, other *corev1.Secret) bool {
	return reflect.DeepEqual(secret.Data, other.Data) && reflect.DeepEqual(secret.Labels, other.Labels) && reflect.DeepEqual(secret.OwnerReferences, other.OwnerReferences)
}

// mustRecreate forces a delete+create for immutable secrets, whose data cannot be updated in place
// and would be rejected by the API server. Checking both current and desired covers the transition
// from a pre-existing mutable secret to an immutable one.
func mustRecreate(current, desired *corev1.Secret) bool {
	return ptr.Deref(current.Immutable, false) || ptr.Deref(desired.Immutable, false)
}
