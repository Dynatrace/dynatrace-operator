//go:build integrationtests

package eec

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/stretchr/testify/require"
)

func TestStatefulSet(t *testing.T) {
	t.Log("WELCOME")

	clt := integrationtests.SetupTestEnvironment(t)

	ctx := context.Background()

	dk := getTestDynakube()

	integrationtests.CreateNamespace(t, ctx, clt, testNamespaceName)
	integrationtests.CreateDynakube(t, ctx, clt, dk)
	mockTLSSecret(t, clt, dk)

	reconciler := NewReconciler(clt, clt, dk)
	err := reconciler.Reconcile(ctx)
	require.NoError(t, err)

	dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true
	err = reconciler.Reconcile(ctx)
	require.NoError(t, err)

	// stop test environment
	integrationtests.DestroyTestEnvironment(t)
}
