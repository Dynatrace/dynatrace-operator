package logmonsettings

import (
	"errors"
	"testing"

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
	ctx := t.Context()

	t.Run("normal run with all scopes and existing setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil)

		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{
			LogMonitoring: &logmonitoring.Spec{},
		}, Status: dynakube.DynaKubeStatus{KubernetesClusterMEID: "meid"}}
		r := &reconciler{dk: dk, dtc: mockClient}

		setReadScope(t, dk)
		setWriteScope(t, dk)

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		verifyCondition(t, dk, alreadyExistReason)
	})

	t.Run("normal run with all scopes and without existing setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)
		mockClient.
			On("CreateLogMonitoringSetting", mock.Anything, "meid", "", mock.Anything).
			Return("test-object-id", nil)

		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{
			LogMonitoring: &logmonitoring.Spec{},
		}, Status: dynakube.DynaKubeStatus{KubernetesClusterMEID: "meid"}}
		r := &reconciler{dk: dk, dtc: mockClient}

		setReadScope(t, dk)
		setWriteScope(t, dk)

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		verifyCondition(t, dk, createdReason)
	})

	t.Run("read-only settings exist -> can not create setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)

		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{
			LogMonitoring: &logmonitoring.Spec{},
		}, Status: dynakube.DynaKubeStatus{KubernetesClusterMEID: "meid"}}
		r := &reconciler{dk: dk, dtc: mockClient}

		setReadScope(t, dk)

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		verifyCondition(t, dk, k8sconditions.OptionalScopeMissingReason)
	})

	t.Run("write-only settings exist -> can not query setting", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)

		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: "meid",
				KubernetesClusterName: "cluster-name",
			},
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{},
			},
		}

		r := &reconciler{dk: dk, dtc: mockClient}

		setWriteScope(t, dk)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		verifyCondition(t, dk, k8sconditions.OptionalScopeMissingReason)
	})
}

func TestCheckLogMonitoringSettings(t *testing.T) {
	ctx := t.Context()

	t.Run("error fetching log monitoring settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(settings.GetSettingsResponse{}, errors.New("error when fetching settings"))

		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: "meid",
			},
		}

		r := &reconciler{
			dk:  dk,
			dtc: mockClient,
		}

		err := r.checkLogMonitoringSettings(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error when fetching settings")

		verifyCondition(t, dk, errorReason)
	})

	t.Run("KubernetesClusterMEID is missing -> skip", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: "",
			},
		}

		r := &reconciler{
			dk:  dk,
			dtc: mockClient,
		}

		err := r.checkLogMonitoringSettings(ctx)
		require.NoError(t, err)

		verifyCondition(t, dk, skippedReason)
	})

	t.Run("log monitoring settings already exist", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil)

		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: "meid",
			},
		}

		r := &reconciler{
			dk:  dk,
			dtc: mockClient,
		}

		err := r.checkLogMonitoringSettings(ctx)
		require.NoError(t, err)

		verifyCondition(t, dk, alreadyExistReason)
	})

	t.Run("create log monitoring settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)
		mockClient.On("CreateLogMonitoringSetting", mock.Anything, "meid", "cluster-name", mock.Anything).
			Return("test-object-id", nil)

		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: "meid",
				KubernetesClusterName: "cluster-name",
			},
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{
					IngestRuleMatchers: []logmonitoring.IngestRuleMatchers{},
				},
			},
		}

		r := &reconciler{
			dk:  dk,
			dtc: mockClient,
		}

		err := r.checkLogMonitoringSettings(ctx)
		require.NoError(t, err)

		verifyCondition(t, dk, createdReason)
	})

	t.Run("error creating log monitoring settings", func(t *testing.T) {
		mockClient := settingsmock.NewAPIClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(settings.GetSettingsResponse{TotalCount: 0}, nil)
		mockClient.On("CreateLogMonitoringSetting", mock.Anything, "meid", "cluster-name", mock.Anything).
			Return("", errors.New("error when creating"))

		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterMEID: "meid",
				KubernetesClusterName: "cluster-name",
			},
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{
					IngestRuleMatchers: []logmonitoring.IngestRuleMatchers{},
				},
			},
		}

		r := &reconciler{
			dk:  dk,
			dtc: mockClient,
		}

		err := r.checkLogMonitoringSettings(ctx)
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
