// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package integrationtests

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateNamespace(t *testing.T, clt client.Client, namespace string) {
	t.Helper()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	create(t, clt, ns)
}

func CreateKubernetesObject(t *testing.T, clt client.Client, object client.Object) {
	t.Helper()

	create(t, clt, object)
}

func CreateDynakube(t *testing.T, clt client.Client, dk *dynakube.DynaKube) {
	t.Helper()

	dkStatus := dk.Status
	create(t, clt, dk)
	dk.Status = dkStatus

	err := dk.UpdateStatus(t.Context(), clt)
	require.NoError(t, err)
}

func create(t *testing.T, clt client.Client, obj client.Object) {
	t.Helper()

	err := clt.Create(t.Context(), obj)
	require.NoError(t, err)

	t.Cleanup(func() {
		// t.Context() is no longer valid on cleanup
		ctx := context.Background()
		require.NoError(t, client.IgnoreNotFound(clt.Delete(context.Background(), obj)))

		// Special handling for namespace deletion
		if ns, ok := obj.(*corev1.Namespace); ok {
			require.NoError(t, client.IgnoreNotFound(clt.Delete(ctx, ns)))
			require.NoError(t, clt.Get(ctx, client.ObjectKeyFromObject(ns), ns))
			ns.Spec.Finalizers = nil
			require.NoError(t, clt.SubResource("finalize").Update(ctx, ns))
		}
	})
}
