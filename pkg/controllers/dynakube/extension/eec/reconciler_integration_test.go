package eec

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/stretchr/testify/require"
)

func TestStatefulSet(t *testing.T) {
	t.Log("WELCOME")

	clt := integrationtests.SetupTestEnvironment(t)

	dk := getTestDynakube()

	integrationtests.CreateNamespace(t, t.Context(), clt, testNamespaceName)
	integrationtests.CreateDynakube(t, t.Context(), clt, dk)
	mockTLSSecret(t, clt, dk)

	reconciler := NewReconciler(clt, clt, dk)
	err := reconciler.Reconcile(t.Context())
	require.NoError(t, err)

	dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true
	err = reconciler.Reconcile(t.Context())
	require.NoError(t, err)
}
