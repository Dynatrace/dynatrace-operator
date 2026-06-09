package logmonitoring

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	oneagentclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/oneagent"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func TestReconcile(t *testing.T) {
	ctx := t.Context()

	t.Run("connection-info fail => error", func(t *testing.T) {
		oaClientMock := oneagentclientmock.NewClient(t)
		dtClient := &dynatrace.Client{OneAgent: oaClientMock}
		dk := &dynakube.DynaKube{}

		failOAConnectionInfo := newMockOaConnectionInfoReconciler(t)
		failOAConnectionInfo.EXPECT().Reconcile(anyCtx, oaClientMock, dk).Return(errors.New("BOOM")).Once()
		r := Reconciler{
			oneAgentConnectionInfoReconciler: failOAConnectionInfo,
		}

		err := r.Reconcile(ctx, dtClient, dk)
		require.Error(t, err)
	})

	t.Run("config-secret fail => error", func(t *testing.T) {
		oaClientMock := oneagentclientmock.NewClient(t)
		dtClient := &dynatrace.Client{OneAgent: oaClientMock}
		dk := &dynakube.DynaKube{}

		passOAConnectionInfo := newMockOaConnectionInfoReconciler(t)
		passOAConnectionInfo.EXPECT().Reconcile(anyCtx, oaClientMock, dk).Return(nil).Once()

		failConfigSecret := newMockSubReconciler(t)
		failConfigSecret.EXPECT().Reconcile(anyCtx, dk).Return(errors.New("BOOM")).Once()

		r := Reconciler{
			oneAgentConnectionInfoReconciler: passOAConnectionInfo,
			configSecretReconciler:           failConfigSecret,
		}

		err := r.Reconcile(ctx, dtClient, dk)
		require.Error(t, err)
	})

	t.Run("all reconcilers pass", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				LogMonitoring: &logmonitoring.Spec{},
			},
		}

		dtClient := &dynatrace.Client{Settings: settings.NewClient(nil)}
		oaClientMock := oneagentclientmock.NewClient(t)
		dtClient.OneAgent = oaClientMock

		passOAConnectionInfo := newMockOaConnectionInfoReconciler(t)
		passOAConnectionInfo.EXPECT().Reconcile(anyCtx, oaClientMock, dk).Return(nil).Once()

		passConfigSecret := newMockSubReconciler(t)
		passConfigSecret.EXPECT().Reconcile(anyCtx, dk).Return(nil).Once()

		passDaemonSet := newMockSubReconciler(t)
		passDaemonSet.EXPECT().Reconcile(anyCtx, dk).Return(nil).Once()

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
