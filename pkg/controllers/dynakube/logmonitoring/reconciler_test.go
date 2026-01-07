package logmonitoring

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestReconcile(t *testing.T) {
	ctx := t.Context()

	t.Run("connection-info fail => error", func(t *testing.T) {
		failOAConnectionInfo := createFailingReconciler(t)
		dk := &dynakube.DynaKube{}
		r := Reconciler{
			dk:                               dk,
			oneAgentConnectionInfoReconciler: failOAConnectionInfo,
		}

		err := r.Reconcile(ctx)
		require.Error(t, err)

		failOAConnectionInfo.AssertCalled(t, "Reconcile", ctx)
	})

	t.Run("config-secret fail => error", func(t *testing.T) {
		failConfigSecret := createFailingReconciler(t)
		passOAConnectionInfo := createPassingReconciler(t)
		dk := &dynakube.DynaKube{}
		r := Reconciler{
			dk:                               dk,
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           failConfigSecret,
		}

		err := r.Reconcile(ctx)
		require.Error(t, err)

		failConfigSecret.AssertCalled(t, "Reconcile", ctx)
		passOAConnectionInfo.AssertCalled(t, "Reconcile", ctx)
	})

	t.Run("all reconcilers pass", func(t *testing.T) {
		passOAConnectionInfo := createPassingReconciler(t)
		passConfigSecret := createPassingReconciler(t)
		passDaemonSet := createPassingReconciler(t)
		passLogMonSetting := createPassingReconciler(t)
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{},
			},
		}
		mockClient := dtclientmock.NewClient(t)
		mockClient.On("GetSettingsForLogModule", mock.AnythingOfType("context.backgroundCtx"), "meid").
			Return(dtclient.GetLogMonSettingsResponse{}, nil).Maybe()
		mockClient.On("CreateLogMonitoringSetting", mock.AnythingOfType("context.backgroundCtx"), "meid", "cluster-name", []logmonitoring.IngestRuleMatchers{}).
			Return("test-object-id", nil).Maybe()

		r := Reconciler{
			dk:                               dk,
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           passConfigSecret,
			daemonsetReconciler:              passDaemonSet,
			logmonsettingsReconciler:         passLogMonSetting,
			dtc:                              mockClient,
		}

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		passConfigSecret.AssertCalled(t, "Reconcile", ctx)
		passOAConnectionInfo.AssertCalled(t, "Reconcile", ctx)
		passDaemonSet.AssertCalled(t, "Reconcile", ctx)
	})
}

func createFailingReconciler(t *testing.T) *controllermock.Reconciler {
	t.Helper()

	failMock := controllermock.NewReconciler(t)
	failMock.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

	return failMock
}

func createPassingReconciler(t *testing.T) *controllermock.Reconciler {
	t.Helper()

	passMock := controllermock.NewReconciler(t)
	passMock.On("Reconcile", mock.Anything).Return(nil)

	return passMock
}
