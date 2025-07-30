package rules

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("no error if not enabled", func(t *testing.T) {
		dk := createDynaKube()
		dk.Spec.MetadataEnrichment.Enabled = ptr.To(false)

		reconciler := NewReconciler(nil, &dk)

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
	})

	t.Run("clean-up if previously enabled", func(t *testing.T) {
		dk := createDynaKube()
		dk.Spec.MetadataEnrichment.Enabled = ptr.To(false)
		dk.Status.MetadataEnrichment.Rules = createRules()
		conditions.SetStatusUpdated(dk.Conditions(), conditionType, "TESTING")

		dtc := dtclientmock.NewClient(t)
		reconciler := NewReconciler(dtc, &dk)

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Empty(t, dk.Status.MetadataEnrichment.Rules)
		assert.Empty(t, dk.Status.Conditions)
	})

	t.Run("no update if not outdated", func(t *testing.T) {
		dk := createDynaKube()
		specialMessage := "TESTING" // if the special message does not change == condition didn't update
		conditions.SetStatusUpdated(dk.Conditions(), conditionType, specialMessage)

		dtc := dtclientmock.NewClient(t)
		reconciler := NewReconciler(dtc, &dk)

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Empty(t, dk.Status.MetadataEnrichment.Rules)
		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, specialMessage, dk.Status.Conditions[0].Message)
	})

	t.Run("update if outdated", func(t *testing.T) {
		dk := createDynaKube()
		conditions.SetScopeAvailable(dk.Conditions(), dtclient.ConditionTypeAPITokenSettingsRead, "available")

		expectedResponse := createRulesResponse()
		specialMessage := "TESTING" // if the special message changes == condition updated
		conditions.SetStatusUpdated(dk.Conditions(), conditionType, specialMessage)

		dtc := dtclientmock.NewClient(t)
		dtc.On("GetRulesSettings", mock.AnythingOfType("context.backgroundCtx"), dk.Status.KubeSystemUUID, dk.Status.KubernetesClusterMEID).Return(expectedResponse, nil)

		futureTime := timeprovider.New()
		futureTime.Set(time.Now().Add(time.Hour))
		reconciler := Reconciler{
			dtc:          dtc,
			dk:           &dk,
			timeProvider: futureTime,
		}

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Equal(t, createRules(), dk.Status.MetadataEnrichment.Rules)
		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		require.NotNil(t, condition)
		assert.NotEqual(t, specialMessage, condition.Message)
	})

	t.Run("set rules correctly", func(t *testing.T) {
		dk := createDynaKube()
		conditions.SetScopeAvailable(dk.Conditions(), dtclient.ConditionTypeAPITokenSettingsRead, "available")

		expectedResponse := createRulesResponse()

		dtc := dtclientmock.NewClient(t)
		dtc.On("GetRulesSettings", mock.AnythingOfType("context.backgroundCtx"), dk.Status.KubeSystemUUID, dk.Status.KubernetesClusterMEID).Return(expectedResponse, nil)
		reconciler := NewReconciler(dtc, &dk)

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Equal(t, createRules(), dk.Status.MetadataEnrichment.Rules)
		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.StatusUpdatedReason, condition.Reason)
	})

	t.Run("set rules correctly, even if only node image pull is set", func(t *testing.T) {
		dk := createDynaKube()
		conditions.SetScopeAvailable(dk.Conditions(), dtclient.ConditionTypeAPITokenSettingsRead, "available")
		dk.Spec.MetadataEnrichment.Enabled = ptr.To(false)

		dk.Annotations = map[string]string{
			exp.OANodeImagePullKey: "true",
		}

		expectedResponse := createRulesResponse()

		dtc := dtclientmock.NewClient(t)
		dtc.On("GetRulesSettings", mock.AnythingOfType("context.backgroundCtx"), dk.Status.KubeSystemUUID, dk.Status.KubernetesClusterMEID).Return(expectedResponse, nil)
		reconciler := NewReconciler(dtc, &dk)

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Equal(t, createRules(), dk.Status.MetadataEnrichment.Rules)
		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.StatusUpdatedReason, condition.Reason)
	})

	t.Run("set api-error condition in case of fail", func(t *testing.T) {
		dk := createDynaKube()
		conditions.SetScopeAvailable(dk.Conditions(), dtclient.ConditionTypeAPITokenSettingsRead, "available")

		dtc := dtclientmock.NewClient(t)
		dtc.On("GetRulesSettings", mock.AnythingOfType("context.backgroundCtx"), dk.Status.KubeSystemUUID, dk.Status.KubernetesClusterMEID).Return(dtclient.GetRulesSettingsResponse{}, errors.New("BOOM"))
		reconciler := NewReconciler(dtc, &dk)

		err := reconciler.Reconcile(ctx)

		require.Error(t, err)
		assert.Empty(t, dk.Status.MetadataEnrichment.Rules)
		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.DynatraceAPIErrorReason, condition.Reason)
	})

	t.Run("no update if optional scope missing", func(t *testing.T) {
		dk := createDynaKube()
		dtc := dtclientmock.NewClient(t)
		reconciler := NewReconciler(dtc, &dk)

		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Empty(t, dk.Status.MetadataEnrichment.Rules)
		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.OptionalScopeMissingReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func createDynaKube() dynakube.DynaKube {
	return dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rules-dk",
		},
		Spec: dynakube.DynaKubeSpec{
			MetadataEnrichment: dynakube.MetadataEnrichment{
				Enabled: ptr.To(true),
			},
		},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID: "kube-system-uuid",
		},
	}
}

func createRulesResponse() dtclient.GetRulesSettingsResponse {
	return dtclient.GetRulesSettingsResponse{
		Items: []dtclient.RuleItem{
			{
				Value: dtclient.RulesResponseValue{
					Rules: createRules(),
				},
			},
		},
	}
}

func createRules() []dynakube.EnrichmentRule {
	return []dynakube.EnrichmentRule{
		{Source: "test1"},
		{Source: "test2"},
	}
}
