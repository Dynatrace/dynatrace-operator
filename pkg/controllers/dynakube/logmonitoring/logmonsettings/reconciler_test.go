package logmonsettings

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/logmonitoring"
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
