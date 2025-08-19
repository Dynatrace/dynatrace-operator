package logmonsettings

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestScopes(t *testing.T) {
	ctx := t.Context()

	t.Run("normal run with all scopes and existing setting", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(dtclient.GetLogMonSettingsResponse{TotalCount: 1}, nil)

		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{
			LogMonitoring: &logmonitoring.Spec{},
		}, Status: dynakube.DynaKubeStatus{KubernetesClusterMEID: "meid"}}
		r := &reconciler{dk: dk, dtc: mockClient}

		setReadScope(t, dk)
		setWriteScope(t, dk)

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		mockClient.AssertCalled(t, "GetSettingsForLogModule", ctx, "meid")
		mockClient.AssertNotCalled(t, "CreateLogMonitoringSetting", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("normal run with all scopes and without existing setting", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(dtclient.GetLogMonSettingsResponse{TotalCount: 0}, nil)
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

		mockClient.AssertCalled(t, "GetSettingsForLogModule", ctx, "meid")
		mockClient.AssertCalled(t, "CreateLogMonitoringSetting", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("read-only settings exist -> can not create setting", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)

		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{
			LogMonitoring: &logmonitoring.Spec{},
		}, Status: dynakube.DynaKubeStatus{KubernetesClusterMEID: "meid"}}
		r := &reconciler{dk: dk, dtc: mockClient}

		setReadScope(t, dk)

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		mockClient.AssertNotCalled(t, "GetSettingsForLogModule", ctx, "meid")
		mockClient.AssertNotCalled(t, "CreateLogMonitoringSetting", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("write-only settings exist -> can not query setting", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)

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

		mockClient.AssertNotCalled(t, "GetSettingsForLogModule", mock.Anything, mock.Anything)
		mockClient.AssertNotCalled(t, "CreateLogMonitoringSetting", ctx, "meid", "cluster-name", mock.Anything)
	})
}

func TestCheckLogMonitoringSettings(t *testing.T) {
	ctx := t.Context()

	t.Run("error fetching log monitoring settings", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(dtclient.GetLogMonSettingsResponse{}, errors.New("error when fetching settings"))

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

		mockClient.AssertCalled(t, "GetSettingsForLogModule", ctx, "meid")
	})

	t.Run("log monitoring settings already exist", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(dtclient.GetLogMonSettingsResponse{TotalCount: 1}, nil)

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

		mockClient.AssertCalled(t, "GetSettingsForLogModule", ctx, "meid")
	})

	t.Run("create log monitoring settings", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(dtclient.GetLogMonSettingsResponse{TotalCount: 0}, nil)
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

		mockClient.AssertCalled(t, "GetSettingsForLogModule", ctx, "meid")
		mockClient.AssertCalled(t, "CreateLogMonitoringSetting", ctx, "meid", "cluster-name", mock.Anything)
	})

	t.Run("error creating log monitoring settings", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(dtclient.GetLogMonSettingsResponse{TotalCount: 0}, nil)
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

		mockClient.AssertCalled(t, "GetSettingsForLogModule", ctx, "meid")
		mockClient.AssertCalled(t, "CreateLogMonitoringSetting", ctx, "meid", "cluster-name", mock.Anything)
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
