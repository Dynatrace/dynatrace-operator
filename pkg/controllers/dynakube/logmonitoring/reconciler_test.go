package logmonitoring

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
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
			oneAgentConnectionInfoReconciler: failOAConnectionInfo,
		}

		err := r.Reconcile(ctx, nil, dk)
		require.Error(t, err)

		failOAConnectionInfo.AssertCalled(t, "Reconcile", ctx)
	})

	t.Run("config-secret fail => error", func(t *testing.T) {
		passOAConnectionInfo := createPassingReconciler(t)
		dk := &dynakube.DynaKube{}

		failConfigSecret := newMockSubReconciler(t)
		failConfigSecret.EXPECT().Reconcile(mock.Anything, dk).Return(errors.New("BOOM")).Once()

		r := Reconciler{
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           failConfigSecret,
		}

		err := r.Reconcile(ctx, nil, dk)
		require.Error(t, err)
	})

	t.Run("all reconcilers pass", func(t *testing.T) {
		passOAConnectionInfo := createPassingReconciler(t)
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{},
			},
		}

		passConfigSecret := newMockSubReconciler(t)
		passConfigSecret.EXPECT().Reconcile(mock.Anything, dk).Return(nil).Once()

		passDaemonSet := newMockSubReconciler(t)
		passDaemonSet.EXPECT().Reconcile(mock.Anything, dk).Return(nil).Once()

		mockDtc := dtclientmock.NewClient(t)
		mockDtc.EXPECT().AsV2().Return(&dtclient.ClientV2{Settings: &settings.Client{}})

		passLogMonSetting := newMockLogmonsettingsSubReconciler(t)
		passLogMonSetting.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()

		r := Reconciler{
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           passConfigSecret,
			daemonsetReconciler:              passDaemonSet,
			logmonsettingsReconciler:         passLogMonSetting,
		}

		err := r.Reconcile(ctx, mockDtc, dk)
		require.NoError(t, err)
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
