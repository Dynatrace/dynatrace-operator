// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/require"
)

func TestStatefulSet(t *testing.T) {
	t.Log("WELCOME")

	clt := integrationtests.SetupTestEnvironment(t)

	ctx := t.Context()

	dk := getTestDynakubeWithTelemetryIngest()

	integrationtests.CreateNamespace(t, ctx, clt, testNamespaceName)
	integrationtests.CreateDynakube(t, ctx, clt, dk)
	mockTLSSecret(t, clt, dk)

	tokenSecret := getTokens(dk.Tokens(), dk.Namespace)
	require.NoError(t, clt.Create(ctx, &tokenSecret))

	configMap := getConfigConfigMap(dk.Name, dk.Namespace)
	require.NoError(t, clt.Create(ctx, &configMap))

	reconciler := NewReconciler(clt, clt)
	err := reconciler.Reconcile(ctx, dk)
	require.NoError(t, err)

	// reconcile again to exercise the update path
	dk.Spec.TelemetryIngest = &telemetryingest.Spec{}
	err = reconciler.Reconcile(ctx, dk)
	require.NoError(t, err)
}
