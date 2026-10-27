// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package integrationtests

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateNamespace(t *testing.T, clt client.Client, namespace string) {
	t.Helper()

	ns := &corev1.Namespace{
		Name: namespace,
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

	if _, isNamespace := obj.(*corev1.Namespace); isNamespace {
		waitForNamespaceAdmission(t, clt, obj.GetName())
	}

	t.Cleanup(func() {
		// t.Context() is no longer valid on cleanup
		ctx := context.Background()
		require.NoError(t, client.IgnoreNotFound(clt.Delete(ctx, obj)))

		// Special handling for namespace deletion
		if ns, isNamespace := obj.(*corev1.Namespace); isNamespace {
			// We need the resourceVersion for the update
			require.NoError(t, clt.Get(ctx, client.ObjectKeyFromObject(ns), ns))
			ns.Spec.Finalizers = nil
			require.NoError(t, clt.SubResource("finalize").Update(ctx, ns))
		}
	})
}

// waitForNamespaceAdmission blocks until the apiserver's NamespaceLifecycle admission plugin accepts creates in the namespace.
// The plugin reads namespaces from an informer cache, which can still hold a previous, terminating namespace
// with the same name and reject creates with "being terminated" even though the new namespace is active.
// A dry-run create goes through admission without persisting anything.
func waitForNamespaceAdmission(t *testing.T, clt client.Client, namespace string) {
	t.Helper()

	require.Eventually(t, func() bool {
		cm := &corev1.ConfigMap{Name: "probe", Namespace: namespace}
		err := clt.Create(t.Context(), cm, client.DryRunAll)

		return !k8serrors.HasStatusCause(err, corev1.NamespaceTerminatingCause)
	}, 10*time.Second, 20*time.Millisecond)
}
