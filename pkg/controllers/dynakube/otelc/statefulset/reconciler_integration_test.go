package statefulset

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/stretchr/testify/require"
)

func TestStatefulSet(t *testing.T) {
	t.Log("WELCOME")

	clt := integrationtests.SetupTestEnvironment(t)

	ctx := context.Background()

	dk := getTestDynakubeWithExtensions()

	integrationtests.CreateNamespace(t, ctx, clt, testNamespaceName)
	integrationtests.CreateDynakube(t, ctx, clt, dk)
	mockTLSSecret(t, clt, dk)

	reconciler := NewReconciler(clt, clt, dk)
	err := reconciler.Reconcile(ctx)
	require.NoError(t, err)

	// enable telemetryIngest
	dk.Spec.TelemetryIngest = &telemetryingest.Spec{}
	err = reconciler.Reconcile(ctx)
	require.NoError(t, err)
}
