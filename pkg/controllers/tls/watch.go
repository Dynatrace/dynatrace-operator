// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/eventfilter"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// WatchCA makes a controller react whenever the operator root CA changes (is created, regenerated, deleted)
// or its bundle changes, so it can renew whatever it derived from the CA, like server certificates.
// mapFn decides which requests to enqueue for such a change.
//
// It watches the CA Secret and the CA bundle ConfigMap in the operator namespace,
// the manager cache has to cover that namespace, which is the case for the Operator manager.
func WatchCA(b *builder.Builder, operatorNamespace string, mapFn handler.MapFunc) *builder.Builder {
	enqueue := handler.EnqueueRequestsFromMapFunc(mapFn)

	return b.
		Watches(&corev1.Secret{}, enqueue,
			builder.WithPredicates(eventfilter.ForObjectNameAndNamespace(CASecretName, operatorNamespace))).
		Watches(&corev1.ConfigMap{}, enqueue,
			builder.WithPredicates(eventfilter.ForObjectNameAndNamespace(CABundleConfigMapName, operatorNamespace)))
}
