package logmonitoring

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	oneagentclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	oneagentclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/oneagent"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })
var anyOAClient = mock.MatchedBy(func(oneagentclient.Client) bool { return true })

func TestReconcile(t *testing.T) {
	ctx := t.Context()

	t.Run("connection-info fail => error", func(t *testing.T) {
		failOAConnectionInfo := newMockOaConnectionInfoReconciler(t)
		failOAConnectionInfo.On("Reconcile", anyCtx, anyOAClient, mock.Anything).Return(errors.New("BOOM"))
		dk := &dynakube.DynaKube{}
		oaClientMock := oneagentclientmock.NewClient(t)
		dtClient := &dynatrace.Client{OneAgent: oaClientMock}
		r := Reconciler{
			oneAgentConnectionInfoReconciler: failOAConnectionInfo,
		}

		err := r.Reconcile(ctx, dtClient, dk)
		require.Error(t, err)

		failOAConnectionInfo.AssertCalled(t, "Reconcile", anyCtx, anyOAClient, dk)
	})

	t.Run("config-secret fail => error", func(t *testing.T) {
		passOAConnectionInfo := newMockOaConnectionInfoReconciler(t)
		passOAConnectionInfo.On("Reconcile", anyCtx, anyOAClient, mock.Anything).Return(nil)
		dk := &dynakube.DynaKube{}

		failConfigSecret := newMockSubReconciler(t)
		failConfigSecret.EXPECT().Reconcile(anyCtx, dk).Return(errors.New("BOOM")).Once()

		oaClientMock := oneagentclientmock.NewClient(t)
		dtClient := &dynatrace.Client{OneAgent: oaClientMock}
		r := Reconciler{
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           failConfigSecret,
		}

		err := r.Reconcile(ctx, dtClient, dk)
		require.Error(t, err)
	})

	t.Run("all reconcilers pass", func(t *testing.T) {
		passOAConnectionInfo := newMockOaConnectionInfoReconciler(t)
		passOAConnectionInfo.On("Reconcile", anyCtx, anyOAClient, mock.Anything).Return(nil)
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

		oaClientMock := oneagentclientmock.NewClient(t)
		dtClient.OneAgent = oaClientMock

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
