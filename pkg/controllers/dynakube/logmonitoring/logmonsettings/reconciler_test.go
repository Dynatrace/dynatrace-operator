package logmonsettings

import (
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	settingsmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcile(t *testing.T) {
	const (
		meID        = "meid"
		clusterName = "cluster-name"
	)

	getDK := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: meID,
				KubernetesClusterName: clusterName,
			},
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{
					IngestRuleMatchers: []logmonitoring.IngestRuleMatchers{},
				},
			},
		}
	}

	t.Run("normal run with all scopes and existing setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetSettingsForLogModule(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil)

		dk := getDK()
		r := NewReconciler(mockClient, dk)

		setReadScope(t, dk)
		setWriteScope(t, dk)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		verifyCondition(t, dk, existsReason)
	})

	t.Run("normal run with all scopes and without existing setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetSettingsForLogModule(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)
		mockClient.EXPECT().CreateLogMonitoringSetting(mock.Anything, meID, clusterName, mock.Anything).
			Return("test-object-id", nil)

		dk := getDK()

		r := NewReconciler(mockClient, dk)

		setReadScope(t, dk)
		setWriteScope(t, dk)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		verifyCondition(t, dk, existsReason)
	})

	t.Run("read-only settings exist -> can not create setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)

		dk := getDK()
		r := NewReconciler(mockClient, dk)

		setReadScope(t, dk)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		verifyCondition(t, dk, k8sconditions.OptionalScopeMissingReason)
	})

	t.Run("write-only settings exist -> can not query setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)

		dk := getDK()

		r := NewReconciler(mockClient, dk)

		setWriteScope(t, dk)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		verifyCondition(t, dk, k8sconditions.OptionalScopeMissingReason)
	})

	t.Run("cleanup condition if logmon is turned off", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)

		dk := &dynakube.DynaKube{}

		r := NewReconciler(mockClient, dk)

		setExistsCondition(dk.Conditions())

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		require.Empty(t, dk.Conditions())
	})

	t.Run("update condition timestamp if outdated", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetSettingsForLogModule(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil)

		dk := getDK()
		setReadScope(t, dk)
		setWriteScope(t, dk)

		r := NewReconciler(mockClient, dk)
		r.timeProvider.Set(time.Now().Add(time.Hour))

		setExistsCondition(dk.Conditions())
		condition := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.NotNil(t, condition)

		prevTS := condition.LastTransitionTime.Time

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		condition = meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.NotNil(t, condition)

		currentTS := condition.LastTransitionTime.Time

		require.NotEqual(t, prevTS, currentTS)
	})

	t.Run("don't update condition timestamp if not outdated", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)

		dk := getDK()

		r := NewReconciler(mockClient, dk)

		setExistsCondition(dk.Conditions())
		condition := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.NotNil(t, condition)

		prevTS := condition.LastTransitionTime.Time

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		condition = meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.NotNil(t, condition)

		currentTS := condition.LastTransitionTime.Time

		require.Equal(t, currentTS, prevTS)
	})
}

func TestCheckLogMonitoringSettings(t *testing.T) {
	const (
		meID        = "meid"
		clusterName = "cluster-name"
	)

	getDK := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: meID,
				KubernetesClusterName: clusterName,
			},
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{
					IngestRuleMatchers: []logmonitoring.IngestRuleMatchers{},
				},
			},
		}
	}

	t.Run("error fetching log monitoring settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetSettingsForLogModule(t.Context(), meID).
			Return(settings.GetSettingsResponse{}, errors.New("error when fetching settings"))

		dk := getDK()

		r := NewReconciler(mockClient, dk)

		err := r.checkLogMonitoringSettings(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "error when fetching settings")

		verifyCondition(t, dk, errorReason)
	})

	t.Run("KubernetesClusterMEID is missing -> skip", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		dk := getDK()
		dk.Status.KubernetesClusterMEID = ""

		r := NewReconciler(mockClient, dk)

		err := r.checkLogMonitoringSettings(t.Context())
		require.NoError(t, err)

		verifyCondition(t, dk, skippedReason)
	})

	t.Run("log monitoring settings already exist", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetSettingsForLogModule(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil)

		dk := getDK()

		r := NewReconciler(mockClient, dk)

		err := r.checkLogMonitoringSettings(t.Context())
		require.NoError(t, err)

		verifyCondition(t, dk, existsReason)
	})

	t.Run("create log monitoring settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetSettingsForLogModule(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)
		mockClient.EXPECT().CreateLogMonitoringSetting(t.Context(), meID, clusterName, mock.Anything).
			Return("test-object-id", nil)

		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: meID,
				KubernetesClusterName: clusterName,
			},
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{
					IngestRuleMatchers: []logmonitoring.IngestRuleMatchers{},
				},
			},
		}

		r := NewReconciler(mockClient, dk)

		err := r.checkLogMonitoringSettings(t.Context())
		require.NoError(t, err)

		verifyCondition(t, dk, existsReason)
	})

	t.Run("error creating log monitoring settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.EXPECT().GetSettingsForLogModule(t.Context(), meID).
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)
		mockClient.EXPECT().CreateLogMonitoringSetting(t.Context(), meID, clusterName, mock.Anything).
			Return("", errors.New("error when creating"))

		dk := getDK()

		r := NewReconciler(mockClient, dk)

		err := r.checkLogMonitoringSettings(t.Context())
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
