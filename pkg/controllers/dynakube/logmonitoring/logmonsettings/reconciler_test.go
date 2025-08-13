package logmonsettings

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckLogMonitoringSettings(t *testing.T) {
	ctx := context.Background()

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

		err := r.checkLogMonitoringSettings(ctx, true, false)
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

		err := r.checkLogMonitoringSettings(ctx, true, false)
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

		err := r.checkLogMonitoringSettings(ctx, true, true)
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

		err := r.checkLogMonitoringSettings(ctx, true, true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error when creating")

		mockClient.AssertCalled(t, "GetSettingsForLogModule", ctx, "meid")
		mockClient.AssertCalled(t, "CreateLogMonitoringSetting", ctx, "meid", "cluster-name", mock.Anything)
	})
	t.Run("read-only settings exist -> can not create setting", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		mockClient.On("GetSettingsForLogModule", mock.Anything, "meid").
			Return(dtclient.GetLogMonSettingsResponse{TotalCount: 1}, nil)

		dk := &dynakube.DynaKube{Status: dynakube.DynaKubeStatus{KubernetesClusterMEID: "meid"}}
		r := &reconciler{dk: dk, dtc: mockClient}

		err := r.checkLogMonitoringSettings(ctx, true, false)
		require.NoError(t, err)

		mockClient.AssertCalled(t, "GetSettingsForLogModule", ctx, "meid")
		mockClient.AssertNotCalled(t, "CreateLogMonitoringSetting", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
	t.Run("write-only settings exist -> can not query setting", func(t *testing.T) {
		mockClient := dtclientmock.NewClient(t)
		mockClient.
			On("CreateLogMonitoringSetting", mock.Anything, "meid", "cluster-name", mock.Anything).
			Return("test-object-id", nil)

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

		err := r.checkLogMonitoringSettings(ctx, false, true)
		require.NoError(t, err)

		mockClient.AssertNotCalled(t, "GetSettingsForLogModule", mock.Anything, mock.Anything)
		mockClient.AssertCalled(t, "CreateLogMonitoringSetting", ctx, "meid", "cluster-name", mock.Anything)
	})
}
