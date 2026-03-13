package logmonitoring

import (
	"context"
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

// mockSubReconciler is a testify mock implementing subReconciler (Reconcile(ctx, dk)).
type mockSubReconciler struct {
	mock.Mock
}

func (m *mockSubReconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	return m.Called(ctx, dk).Error(0)
}

// mockLogmonsettingsSubReconciler is a testify mock implementing logmonsettingsSubReconciler.
type mockLogmonsettingsSubReconciler struct {
	mock.Mock
}

func (m *mockLogmonsettingsSubReconciler) Reconcile(ctx context.Context, dtc settings.APIClient, dk *dynakube.DynaKube) error {
	return m.Called(ctx, dtc, dk).Error(0)
}

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
		failConfigSecret := createFailingSubReconciler(t)
		passOAConnectionInfo := createPassingReconciler(t)
		dk := &dynakube.DynaKube{}
		r := Reconciler{
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           failConfigSecret,
		}

		err := r.Reconcile(ctx, nil, dk)
		require.Error(t, err)

		passOAConnectionInfo.AssertCalled(t, "Reconcile", ctx)
		failConfigSecret.AssertCalled(t, "Reconcile", ctx, dk)
	})

	t.Run("all reconcilers pass", func(t *testing.T) {
		passOAConnectionInfo := createPassingReconciler(t)
		passConfigSecret := createPassingSubReconciler(t)
		passDaemonSet := createPassingSubReconciler(t)
		passLogMonSetting := createPassingLogmonsettingsSubReconciler(t)
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{},
			},
		}

		mockDtc := dtclientmock.NewClient(t)
		mockDtc.EXPECT().AsV2().Return(&dtclient.ClientV2{Settings: &settings.Client{}})

		r := Reconciler{
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           passConfigSecret,
			daemonsetReconciler:              passDaemonSet,
			logmonsettingsReconciler:         passLogMonSetting,
		}

		err := r.Reconcile(ctx, mockDtc, dk)
		require.NoError(t, err)

		passOAConnectionInfo.AssertCalled(t, "Reconcile", ctx)
		passConfigSecret.AssertCalled(t, "Reconcile", ctx, dk)
		passDaemonSet.AssertCalled(t, "Reconcile", ctx, dk)
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

func createFailingSubReconciler(t *testing.T) *mockSubReconciler {
	t.Helper()

	m := &mockSubReconciler{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	m.On("Reconcile", mock.Anything, mock.Anything).Return(errors.New("BOOM"))

	return m
}

func createPassingSubReconciler(t *testing.T) *mockSubReconciler {
	t.Helper()

	m := &mockSubReconciler{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	m.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

	return m
}

func createPassingLogmonsettingsSubReconciler(t *testing.T) *mockLogmonsettingsSubReconciler {
	t.Helper()

	m := &mockLogmonsettingsSubReconciler{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	m.On("Reconcile", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return m
}
