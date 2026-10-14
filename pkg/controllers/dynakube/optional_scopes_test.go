// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	tokenclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/k8sentity"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	settingsmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/settings"
	tokenclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/utils/ptr"
)

const testKubeSystemUUID = "kube-system-uuid"

var (
	requiredScopes = []string{
		tokenclient.ScopeDataExport,
		tokenclient.ScopeInstallerDownload,
		tokenclient.ScopeActiveGateTokenCreate,
	}
	allScopes = append([]string{tokenclient.ScopeSettingsRead, tokenclient.ScopeSettingsWrite}, requiredScopes...)
)

func newRegistrationDisabledDynaKube() *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		Name:      testName,
		Namespace: testNamespace,
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
		},
	}
	dk.Status.KubeSystemUUID = testKubeSystemUUID

	return dk
}

func TestScopesWithKubeMonRegistrationDisabled(t *testing.T) {
	t.Run("kubemon without registration", func(t *testing.T) {
		dk := newRegistrationDisabledDynaKube()
		dk.Spec.KubernetesMonitoring = &kubemon.Spec{}

		settingsClient := settingsmock.NewClient(t)
		settingsClient.EXPECT().GetK8sClusterME(anyCtx, testKubeSystemUUID).Return(settings.K8sClusterME{}, nil).Once()

		reconcileTokensAndK8sEntity(t, dk, allScopes, settingsClient)

		assertOptionalScopesAvailable(t, dk)
		assertMEIDConditionNotScopeMissing(t, dk)
	})

	t.Run("kubernetes-monitoring with automatic-kubernetes-api-monitoring disabled", func(t *testing.T) {
		dk := newRegistrationDisabledDynaKube()
		dk.Annotations = map[string]string{exp.AGAutomaticK8sAPIMonitoringKey: "false"}
		dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}

		settingsClient := settingsmock.NewClient(t)
		settingsClient.EXPECT().GetK8sClusterME(anyCtx, testKubeSystemUUID).Return(settings.K8sClusterME{}, nil).Once()

		reconcileTokensAndK8sEntity(t, dk, allScopes, settingsClient)

		assertOptionalScopesAvailable(t, dk)
		assertMEIDConditionNotScopeMissing(t, dk)
	})

	t.Run("token without settings scopes still reports ScopeMissing", func(t *testing.T) {
		dk := newRegistrationDisabledDynaKube()
		dk.Spec.KubernetesMonitoring = &kubemon.Spec{}

		reconcileTokensAndK8sEntity(t, dk, requiredScopes, settingsmock.NewClient(t))
		assert.False(t, ptr.Deref(dk.Status.APIToken.AvailableOptionalScopes.SettingsRead, true))
		assert.False(t, ptr.Deref(dk.Status.APIToken.AvailableOptionalScopes.SettingsWrite, true))

		condition := meta.FindStatusCondition(*dk.Conditions(), "MonitoredEntity")
		require.NotNil(t, condition)
		assert.Equal(t, k8sconditions.OptionalScopeMissingReason, condition.Reason)
	})
}

func reconcileTokensAndK8sEntity(t *testing.T, dk *dynakube.DynaKube, tokenScopes []string, settingsClient *settingsmock.Client) {
	t.Helper()

	tokenClient := tokenclientmock.NewClient(t)
	tokenClient.EXPECT().GetScopes(anyCtx, testAPIToken).Return(tokenScopes, nil)

	fakeClient := fake.NewClient(createAPISecret())
	controller := &Controller{
		client:              fakeClient,
		apiReader:           fakeClient,
		dtClientFactory:     newClientFactory(&dynatrace.Client{Token: tokenClient, Settings: settingsClient}),
		k8sEntityReconciler: k8sentity.NewReconciler(),
	}

	dtClient, err := controller.setupTokensAndClient(t.Context(), dk)
	require.NoError(t, err)

	err = controller.k8sEntityReconciler.Reconcile(t.Context(), dtClient.Settings, dk)
	require.NoError(t, err)
}

func assertOptionalScopesAvailable(t *testing.T, dk *dynakube.DynaKube) {
	t.Helper()

	assert.True(t, ptr.Deref(dk.Status.APIToken.AvailableOptionalScopes.SettingsRead, false))
	assert.True(t, ptr.Deref(dk.Status.APIToken.AvailableOptionalScopes.SettingsWrite, false))
}

func assertMEIDConditionNotScopeMissing(t *testing.T, dk *dynakube.DynaKube) {
	t.Helper()

	condition := meta.FindStatusCondition(*dk.Conditions(), "MonitoredEntity")
	require.NotNil(t, condition)
	assert.NotEqual(t, k8sconditions.OptionalScopeMissingReason, condition.Reason)
	assert.Empty(t, dk.Status.KubernetesClusterMEID)
}
