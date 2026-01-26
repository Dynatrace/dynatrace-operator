package settings

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	settingsmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcile(t *testing.T) {
	const meID = "meid"
	getDK := func(withKSPM bool) *dynakube.DynaKube {
		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				},
			},
		}, Status: dynakube.DynaKubeStatus{KubernetesClusterMEID: meID}}

		if withKSPM {
			dk.Spec.Kspm = &kspm.Spec{}
		}

		return dk
	}

	t.Run("normal run with all scopes and existing setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetKSPMSettings(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil)

		dk := getDK(true)
		r := NewReconciler()

		setReadScope(t, dk)
		setWriteScope(t, dk)

		err := r.Reconcile(t.Context(), mockClient, dk)
		require.NoError(t, err)

		verifyCondition(t, dk, alreadyExistReason)
	})

	t.Run("normal run with all scopes and without existing setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetKSPMSettings(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)

		mockClient.EXPECT().CreateKSPMSetting(t.Context(), meID, true).
			Return("test-object-id", nil)

		dk := getDK(true)
		r := NewReconciler()

		setReadScope(t, dk)
		setWriteScope(t, dk)

		err := r.Reconcile(t.Context(), mockClient, dk)
		require.NoError(t, err)

		verifyCondition(t, dk, createdReason)
	})

	t.Run("read-only settings exist -> can not create setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)

		dk := getDK(true)
		r := NewReconciler()

		setReadScope(t, dk)

		err := r.Reconcile(t.Context(), mockClient, dk)
		require.NoError(t, err)

		verifyCondition(t, dk, k8sconditions.OptionalScopeMissingReason)
	})

	t.Run("write-only settings exist -> can not query setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)

		dk := getDK(true)

		r := NewReconciler()

		setWriteScope(t, dk)

		err := r.Reconcile(t.Context(), mockClient, dk)
		require.NoError(t, err)

		verifyCondition(t, dk, k8sconditions.OptionalScopeMissingReason)
	})

	t.Run("create setting without KSPM", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetKSPMSettings(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)

		mockClient.EXPECT().CreateKSPMSetting(t.Context(), meID, false).
			Return("test-object-id", nil)

		dk := getDK(false)

		r := NewReconciler()

		setReadScope(t, dk)
		setWriteScope(t, dk)

		err := r.Reconcile(t.Context(), mockClient, dk)
		require.NoError(t, err)

		verifyCondition(t, dk, createdReason)
	})

	t.Run("cleanup condition if KubeMon is turned off", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)

		dk := &dynakube.DynaKube{}

		r := NewReconciler()

		setCreatedCondition(dk.Conditions(), false)

		err := r.Reconcile(t.Context(), mockClient, dk)
		require.NoError(t, err)

		require.Empty(t, dk.Conditions())
	})
}

func TestCheckKSPMSettings(t *testing.T) {
	const meID = "meid"
	getDK := func(withKSPM bool, meID string) *dynakube.DynaKube {
		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				},
			},
		}, Status: dynakube.DynaKubeStatus{KubernetesClusterMEID: meID}}

		if withKSPM {
			dk.Spec.Kspm = &kspm.Spec{}
		}

		return dk
	}

	t.Run("error fetching kspm settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetKSPMSettings(t.Context(), meID).
			Return(settings.GetSettingsResponse{}, errors.New("error when fetching settings"))

		dk := getDK(true, meID)

		r := NewReconciler()

		err := r.checkKSPMSettings(t.Context(), mockClient, dk)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error when fetching settings")

		verifyCondition(t, dk, errorReason)
	})

	t.Run("KubernetesClusterMEID is missing -> skip", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		dk := getDK(true, "")

		r := NewReconciler()

		err := r.checkKSPMSettings(t.Context(), mockClient, dk)
		require.NoError(t, err)

		verifyCondition(t, dk, skippedReason)
	})

	t.Run("kspm settings already exist", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetKSPMSettings(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil)

		dk := getDK(false, meID)

		r := NewReconciler()

		err := r.checkKSPMSettings(t.Context(), mockClient, dk)
		require.NoError(t, err)

		verifyCondition(t, dk, alreadyExistReason)
	})

	t.Run("create kspm settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetKSPMSettings(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)
		mockClient.EXPECT().CreateKSPMSetting(t.Context(), meID, true).
			Return("test-object-id", nil)

		dk := getDK(true, meID)

		r := NewReconciler()

		err := r.checkKSPMSettings(t.Context(), mockClient, dk)
		require.NoError(t, err)

		verifyCondition(t, dk, createdReason)
	})

	t.Run("create kubemon-only settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetKSPMSettings(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)
		mockClient.EXPECT().CreateKSPMSetting(t.Context(), meID, false).
			Return("test-object-id", nil)

		dk := getDK(false, meID)

		r := NewReconciler()

		err := r.checkKSPMSettings(t.Context(), mockClient, dk)
		require.NoError(t, err)

		verifyCondition(t, dk, createdReason)
	})

	t.Run("error creating kspm settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetKSPMSettings(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)
		mockClient.EXPECT().CreateKSPMSetting(t.Context(), meID, true).
			Return("", errors.New("error when creating"))

		dk := getDK(true, meID)

		r := NewReconciler()

		err := r.checkKSPMSettings(t.Context(), mockClient, dk)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error when creating")

		verifyCondition(t, dk, errorReason)
	})
}

func setReadScope(t *testing.T, dk *dynakube.DynaKube) {
	t.Helper()
	meta.SetStatusCondition(dk.Conditions(), metav1.Condition{Type: dtclient.ConditionTypeAPITokenSettingsRead, Status: metav1.ConditionTrue})
}

func setWriteScope(t *testing.T, dk *dynakube.DynaKube) {
	t.Helper()
	meta.SetStatusCondition(dk.Conditions(), metav1.Condition{Type: dtclient.ConditionTypeAPITokenSettingsWrite, Status: metav1.ConditionTrue})
}

func verifyCondition(t *testing.T, dk *dynakube.DynaKube, expectedReason string) {
	t.Helper()

	c := meta.FindStatusCondition(*dk.Conditions(), ConditionType)

	require.NotNil(t, c)
	assert.Equal(t, expectedReason, c.Reason)
}
