package apimonitoring

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	settingsmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testUID      = "test-uid"
	testName     = "test-clusterLabel"
	testObjectID = "test-objectid"
	testMEID     = "test-MEID"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

type testReconciler struct {
	Reconciler

	dtClient *settingsmock.APIClient
}

func newTestReconciler(t *testing.T, dk *dynakube.DynaKube) *testReconciler {
	dtClient := settingsmock.NewAPIClient(t)

	return &testReconciler{
		Reconciler: Reconciler{
			dtc:          dtClient,
			dk:           dk,
			clusterLabel: testName,
		},
		dtClient: dtClient,
	}
}

func createMonitoredEntities() settings.K8sClusterME {
	return settings.K8sClusterME{
		ID: "KUBERNETES_CLUSTER-119C75CCDA94799F", Name: "operator test entity 1",
	}
}

func TestReconcile(t *testing.T) {
	t.Run("create setting when no settings for the found monitored entities are existing", func(t *testing.T) {
		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{ID: testMEID}, settings.KubernetesSettingsSchemaID).
			Return(settings.GetSettingsResponse{}, nil).Once()
		r.dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, testName, testUID, testMEID).
			Return(testObjectID, nil).Once()

		actual, err := r.createObjectIDIfNotExists(t.Context())
		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run("don't create setting when settings for the found monitored entities are existing", func(t *testing.T) {
		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{ID: testMEID}, settings.KubernetesSettingsSchemaID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil).Once()
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{ID: testMEID}, settings.AppTransitionSchemaID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil).Once()

		actual, err := r.createObjectIDIfNotExists(t.Context())
		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("optional scope settings.read not available", func(t *testing.T) {
		dk := newDynaKube()
		k8sconditions.SetOptionalScopeMissing(dk.Conditions(), dtclient.ConditionTypeAPITokenSettingsRead, "not available")
		r := newTestReconciler(t, dk)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)
	})

	t.Run("optional scope settings.write not available", func(t *testing.T) {
		dk := newDynaKube()
		k8sconditions.SetOptionalScopeMissing(dk.Conditions(), dtclient.ConditionTypeAPITokenSettingsWrite, "not available")
		r := newTestReconciler(t, dk)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)
	})

	t.Run("app-transition schema not found", func(t *testing.T) {
		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{ID: testMEID}, settings.KubernetesSettingsSchemaID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil).Once()
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{ID: testMEID}, settings.AppTransitionSchemaID).
			Return(settings.GetSettingsResponse{}, &core.HTTPError{StatusCode: 404}).Once()

		err := r.Reconcile(t.Context())
		require.NoError(t, err)
	})

	t.Run("reconcile k8sentity again on first run", func(t *testing.T) {
		dk := newDynaKube()
		dk.Status.KubernetesClusterMEID = ""
		r := newTestReconciler(t, dk)
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{}, settings.KubernetesSettingsSchemaID).
			Return(settings.GetSettingsResponse{}, nil).Once()
		r.dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, testName, testUID, "").
			Return(testObjectID, nil).Once()

		actual, err := r.createObjectIDIfNotExists(t.Context())
		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})
}

func TestReconcileErrors(t *testing.T) {
	t.Run("missing kube-system uuid", func(t *testing.T) {
		dk := newDynaKube()
		dk.Status.KubeSystemUUID = ""
		r := newTestReconciler(t, dk)

		actual, err := r.createObjectIDIfNotExists(t.Context())

		require.ErrorIs(t, err, errMissingKubeSystemUUID)
		assert.Empty(t, actual)
	})

	t.Run("get kubernetes settings fails", func(t *testing.T) {
		expectErr := errors.New("could not get settings for monitored entities")

		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{ID: testMEID}, settings.KubernetesSettingsSchemaID).
			Return(settings.GetSettingsResponse{}, expectErr).Once()

		actual, err := r.createObjectIDIfNotExists(t.Context())

		require.ErrorIs(t, err, expectErr)
		assert.Empty(t, actual)
	})

	t.Run("create kubernetes setting fails", func(t *testing.T) {
		expectErr := errors.New("could not create kubernetes setting")

		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{ID: testMEID}, settings.KubernetesSettingsSchemaID).
			Return(settings.GetSettingsResponse{}, nil).Once()
		r.dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, testName, testUID, testMEID).
			Return("", expectErr).Once()

		actual, err := r.createObjectIDIfNotExists(t.Context())

		require.ErrorIs(t, err, expectErr)
		assert.Empty(t, actual)
	})

	t.Run("create kubernetes app setting fails", func(t *testing.T) {
		expectErr := errors.New("could not create monitored entity")

		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{ID: testMEID}, settings.KubernetesSettingsSchemaID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil).Once()
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{ID: testMEID}, settings.AppTransitionSchemaID).
			Return(settings.GetSettingsResponse{}, nil).Once()
		r.dtClient.EXPECT().
			CreateOrUpdateKubernetesAppSetting(anyCtx, testMEID).
			Return("", expectErr).Once()

		actual, err := r.createObjectIDIfNotExists(t.Context())

		require.ErrorIs(t, err, expectErr)
		assert.Empty(t, actual)
	})
}

func TestHandleKubernetesAppEnabled(t *testing.T) {
	t.Run("don't create app setting due to empty MonitoredEntitys", func(t *testing.T) {
		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{}, settings.AppTransitionSchemaID).
			Return(settings.GetSettingsResponse{}, nil).Once()

		err := r.handleKubernetesAppEnabled(t.Context(), settings.K8sClusterME{})
		require.NoError(t, err)
	})

	t.Run("don't create app setting as settings already exist", func(t *testing.T) {
		entity := createMonitoredEntities()
		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, entity, settings.AppTransitionSchemaID).
			Return(settings.GetSettingsResponse{TotalCount: 1}, nil).Once()

		err := r.handleKubernetesAppEnabled(t.Context(), entity)
		require.NoError(t, err)
	})

	t.Run("don't create app setting when get entities api response is error", func(t *testing.T) {
		expectErr := errors.New("could not get monitored entities")

		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, settings.K8sClusterME{}, settings.AppTransitionSchemaID).
			Return(settings.GetSettingsResponse{}, expectErr).Once()

		err := r.handleKubernetesAppEnabled(t.Context(), settings.K8sClusterME{})
		require.ErrorIs(t, err, expectErr)
	})

	t.Run("don't create app setting when get CreateOrUpdateKubernetesAppSetting response is error", func(t *testing.T) {
		expectErr := errors.New("could not create app setting")

		entity := createMonitoredEntities()
		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, entity, settings.AppTransitionSchemaID).
			Return(settings.GetSettingsResponse{}, nil).Once()
		r.dtClient.EXPECT().
			CreateOrUpdateKubernetesAppSetting(anyCtx, entity.ID).
			Return("", expectErr).Once()

		err := r.handleKubernetesAppEnabled(t.Context(), entity)
		require.ErrorIs(t, err, expectErr)
	})

	t.Run("create app setting as settings already exist", func(t *testing.T) {
		entity := createMonitoredEntities()
		r := newTestReconciler(t, newDynaKube())
		r.dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, entity, settings.AppTransitionSchemaID).
			Return(settings.GetSettingsResponse{}, nil).Once()
		r.dtClient.EXPECT().
			CreateOrUpdateKubernetesAppSetting(anyCtx, entity.ID).
			Return("test", nil).Once()

		err := r.handleKubernetesAppEnabled(t.Context(), entity)
		require.NoError(t, err)
	})
}

func newDynaKube() *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-oneagent",
			Namespace: "my-namespace",
			UID:       "69e98f18-805a-42de-84b5-3eae66534f75",
			Annotations: map[string]string{
				exp.AGK8sAppEnabledKey: "true",
			},
		},
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				HostMonitoring: &oneagent.HostInjectSpec{},
			},
		},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID:        testUID,
			KubernetesClusterMEID: testMEID,
		},
	}
	k8sconditions.SetOptionalScopeAvailable(&dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsRead, "available")
	k8sconditions.SetOptionalScopeAvailable(&dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsWrite, "available")

	return dk
}

func Test_logCache(t *testing.T) {
	reset := func() {
		logCache = make(map[string]time.Time)
	}

	t.Run("lookup", func(t *testing.T) {
		t.Cleanup(reset)

		fixedTime := time.Now()
		timeNow = func() time.Time { return fixedTime }

		assert.True(t, shouldLogMissingAppTransitionSchema("test"))
		assert.False(t, shouldLogMissingAppTransitionSchema("test"))

		// advance time to just before timeout
		fixedTime = fixedTime.Add(logCacheTimeout)
		assert.False(t, shouldLogMissingAppTransitionSchema("test"))
		// advance time to after timeout
		fixedTime = fixedTime.Add(1 * time.Second)
		assert.True(t, shouldLogMissingAppTransitionSchema("test"))
	})

	t.Run("limit size", func(t *testing.T) {
		t.Cleanup(reset)

		for i := range 101 {
			require.True(t, shouldLogMissingAppTransitionSchema(strconv.Itoa(i)))
		}

		require.Len(t, logCache, 100)
		require.Contains(t, logCache, "99")
		require.NotContains(t, logCache, "100")
	})
}
