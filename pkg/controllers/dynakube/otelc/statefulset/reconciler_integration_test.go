// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	imageclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/require"
)

func TestStatefulSet(t *testing.T) {
	t.Log("WELCOME")

	clt := integrationtests.SetupTestEnvironment(t)

	ctx := t.Context()

	dk := getTestDynakubeWithTelemetryIngest()

	integrationtests.CreateNamespace(t, clt, testNamespaceName)
	integrationtests.CreateDynakube(t, clt, dk)
	mockTLSSecret(t, clt, dk)

	tokenSecret := getTokens(dk.Tokens(), dk.Namespace)
	require.NoError(t, clt.Create(ctx, &tokenSecret))

	configMap := getConfigConfigMap(dk.Name, dk.Namespace)
	require.NoError(t, clt.Create(ctx, &configMap))

	reconciler := NewReconciler(clt, clt)
	err := reconciler.Reconcile(ctx, imageclientmock.NewClient(t), dk)
	require.NoError(t, err)

	// reconcile again to exercise the update path
	dk.Spec.TelemetryIngest = &telemetryingest.Spec{}
	err = reconciler.Reconcile(ctx, imageclientmock.NewClient(t), dk)
	require.NoError(t, err)
}
