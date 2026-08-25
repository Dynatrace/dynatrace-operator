// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package eec

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/require"
)

func TestStatefulSet(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)

	dk := getTestDynakube()

	integrationtests.CreateNamespace(t, clt, testNamespaceName)
	integrationtests.CreateDynakube(t, clt, dk)
	mockTLSSecret(t, clt, dk)

	reconciler := NewReconciler(clt, clt)
	err := reconciler.Reconcile(t.Context(), nil, dk)
	require.NoError(t, err)

	dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = new(true)
	err = reconciler.Reconcile(t.Context(), nil, dk)
	require.NoError(t, err)
}
