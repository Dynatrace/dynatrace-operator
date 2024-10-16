package logmodule

import (
	"context"
	"errors"
	"testing"

	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("connection-info fail => error", func(t *testing.T) {
		failOAConnectionInfo := createFailingReconciler(t)
		r := Reconciler{
			oneAgentConnectionInfoReconciler: failOAConnectionInfo,
		}

		err := r.Reconcile(ctx)
		require.Error(t, err)

		failOAConnectionInfo.AssertCalled(t, "Reconcile", ctx)
	})

	t.Run("config-secret fail => error", func(t *testing.T) {
		failConfigSecret := createFailingReconciler(t)
		passOAConnectionInfo := createPassingReconciler(t)
		r := Reconciler{
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
		r := Reconciler{
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           passConfigSecret,
			daemonsetReconciler:              passDaemonSet,
		}

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		passConfigSecret.AssertCalled(t, "Reconcile", ctx)
		passOAConnectionInfo.AssertCalled(t, "Reconcile", ctx)
	})
}

func createFailingReconciler(t *testing.T) *controllermock.Reconciler {
	failMock := controllermock.NewReconciler(t)
	failMock.On("Reconcile", mock.Anything).Return(errors.New("BOOM"))

	return failMock
}

func createPassingReconciler(t *testing.T) *controllermock.Reconciler {
	passMock := controllermock.NewReconciler(t)
	passMock.On("Reconcile", mock.Anything).Return(nil)

	return passMock
}
