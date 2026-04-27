package logmonitoring

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

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

		failOAConnectionInfo.AssertCalled(t, "Reconcile", anyCtx)
	})

	t.Run("config-secret fail => error", func(t *testing.T) {
		passOAConnectionInfo := createPassingReconciler(t)
		dk := &dynakube.DynaKube{}

		failConfigSecret := newMockSubReconciler(t)
		failConfigSecret.EXPECT().Reconcile(anyCtx, dk).Return(errors.New("BOOM")).Once()

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
		passConfigSecret.EXPECT().Reconcile(anyCtx, dk).Return(nil).Once()

		passDaemonSet := newMockSubReconciler(t)
		passDaemonSet.EXPECT().Reconcile(anyCtx, dk).Return(nil).Once()

		dtClient := &dynatrace.Client{Settings: settings.NewClient(nil)}

		passLogMonSetting := newMockLogmonsettingsSubReconciler(t)
		passLogMonSetting.EXPECT().Reconcile(anyCtx, dtClient.Settings, dk).Return(nil).Once()

		r := Reconciler{
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           passConfigSecret,
			daemonsetReconciler:              passDaemonSet,
			logmonsettingsReconciler:         passLogMonSetting,
		}

		err := r.Reconcile(ctx, dtClient, dk)
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
