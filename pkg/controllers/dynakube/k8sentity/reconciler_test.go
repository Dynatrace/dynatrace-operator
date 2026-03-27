package k8sentity

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	settingsmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func newDynaKube() *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-oneagent",
			Namespace:   "my-namespace",
			Annotations: map[string]string{},
		},
	}

	k8sconditions.SetOptionalScopeAvailable(&dk.Status.Conditions, token.ConditionTypeAPITokenSettingsRead, "available")
	k8sconditions.SetOptionalScopeAvailable(&dk.Status.Conditions, token.ConditionTypeAPITokenSettingsWrite, "available")

	return dk
}

func enableKubeMon(dk *dynakube.DynaKube) {
	dk.Spec.ActiveGate = activegate.Spec{
		Capabilities: []activegate.CapabilityDisplayName{
			activegate.KubeMonCapability.DisplayName,
		},
	}
}

func setSystemUUID(dk *dynakube.DynaKube, uuid string) {
	dk.Status.KubeSystemUUID = uuid
}

func setClusterNameFF(dk *dynakube.DynaKube, name string) {
	dk.Annotations[exp.AGAutomaticK8sAPIMonitoringClusterNameKey] = name
}

func enableAppFF(dk *dynakube.DynaKube) {
	dk.Annotations[exp.AGK8sAppEnabledKey] = "true"
}

func setMEInfo(dk *dynakube.DynaKube, me settings.K8sClusterME) {
	dk.Status.KubernetesClusterMEID = me.ID
	dk.Status.KubernetesClusterName = me.Name
}

func setCondition(dk *dynakube.DynaKube) {
	k8sconditions.SetStatusUpdated(&dk.Status.Conditions, meIDConditionType, "Kubernetes Cluster MEID is up to date")
}

func TestReconcile(t *testing.T) {
	const (
		meID       = "KUBERNETES_CLUSTER-119C75CCDA94799F"
		systemUUID = "2132143215"
	)
	t.Run("optional scope settings.read not available", func(t *testing.T) {
		dk := newDynaKube()
		k8sconditions.SetOptionalScopeMissing(dk.Conditions(), token.ConditionTypeAPITokenSettingsRead, "not available")

		r := NewReconciler()
		err := r.Reconcile(t.Context(), settingsmock.NewAPIClient(t), dk)
		require.NoError(t, err)
	})

	t.Run("missing kube-system uuid", func(t *testing.T) {
		dk := newDynaKube()

		r := NewReconciler()
		err := r.Reconcile(t.Context(), settingsmock.NewAPIClient(t), dk)

		require.ErrorIs(t, err, errMissingKubeSystemUUID)
	})

	t.Run("refreshes MEID immediately after creating settings on first run", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: dk.Name}
		enableKubeMon(dk)
		enableAppFF(dk)
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewAPIClient(t)
		// 1. reconcileClusterMEID: no ME found yet (settings object not yet created)
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(settings.K8sClusterME{}, nil).Once() // this Once() is needed, otherwise the second EXPECT for this func will do nothing -> test fails
		// 2. createObjectIDIfNotExists: no settings exist yet, create them
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, me.Name, systemUUID, "").
			Return(me.ID, nil)
		// 3. refreshClusterMEID: ME is now available after settings were created
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(me, nil)
		// 4. reconcile app transition schema
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{}, nil)
		dtClient.EXPECT().
			CreateOrUpdateKubernetesAppSetting(anyCtx, me.ID).
			Return("some-id", nil).Once()

		r := NewReconciler()
		err := r.Reconcile(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, me.ID, dk.Status.KubernetesClusterMEID)
		assert.Equal(t, me.Name, dk.Status.KubernetesClusterName)
		assert.NotNil(t, meta.FindStatusCondition(*dk.Conditions(), meIDConditionType))
	})

	t.Run("only MEID reconcile without AG", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: dk.Name}
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewAPIClient(t)
		// 1. reconcileClusterMEID: no ME found yet (settings object not yet created)
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(me, nil).Once() // this Once() is needed, otherwise the second EXPECT for this func will do nothing -> test fails
		r := NewReconciler()
		err := r.Reconcile(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, me.ID, dk.Status.KubernetesClusterMEID)
		assert.Equal(t, me.Name, dk.Status.KubernetesClusterName)
		assert.NotNil(t, meta.FindStatusCondition(*dk.Conditions(), meIDConditionType))
	})

	t.Run("no app setting reconcile without FF", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: dk.Name}
		enableKubeMon(dk)
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewAPIClient(t)
		// 1. reconcileClusterMEID: no ME found yet (settings object not yet created)
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(settings.K8sClusterME{}, nil).Once() // this Once() is needed, otherwise the second EXPECT for this func will do nothing -> test fails
		// 2. createObjectIDIfNotExists: no settings exist yet, create them
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, me.Name, systemUUID, "").
			Return(me.ID, nil)
		// 3. refreshClusterMEID: ME is now available after settings were created
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(me, nil)

		r := NewReconciler()
		err := r.Reconcile(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, me.ID, dk.Status.KubernetesClusterMEID)
		assert.Equal(t, me.Name, dk.Status.KubernetesClusterName)
		assert.NotNil(t, meta.FindStatusCondition(*dk.Conditions(), meIDConditionType))
	})
}

func TestReconcileMEID(t *testing.T) {
	const (
		meID       = "KUBERNETES_CLUSTER-119C75CCDA94799F"
		meName     = "my-cluster"
		systemUUID = "2132143215"
	)

	t.Run("skipped when MEID condition is up to date", func(t *testing.T) {
		dk := newDynaKube()
		setCondition(dk)

		dtClient := settingsmock.NewAPIClient(t)
		r := NewReconciler()
		err := r.reconcileMEID(t.Context(), dtClient, dk)
		require.NoError(t, err)
	})

	t.Run("sets MEID when ME is found", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: meName}
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(me, nil)

		r := NewReconciler()
		err := r.reconcileMEID(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, me.ID, dk.Status.KubernetesClusterMEID)
		assert.Equal(t, me.Name, dk.Status.KubernetesClusterName)
	})

	t.Run("no error if no MEs are found", func(t *testing.T) {
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)
		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(settings.K8sClusterME{}, nil)

		r := NewReconciler()
		err := r.reconcileMEID(t.Context(), dtClient, dk)
		require.NoError(t, err)
	})
}

func TestCreateK8sConnectionSettingIfAbsent(t *testing.T) {
	const (
		meID       = "KUBERNETES_CLUSTER-119C75CCDA94799F"
		systemUUID = "2132143215"
		objectID   = "2141rfa3sjvnsk"
	)

	t.Run("don't create setting when settings when MEID is already in dk", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: dk.Name}
		setSystemUUID(dk, systemUUID)
		setMEInfo(dk, me)

		dtClient := settingsmock.NewAPIClient(t)

		r := NewReconciler()
		actual, err := r.createK8sConnectionSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Empty(t, actual)
	})
	t.Run("creates setting", func(t *testing.T) {
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, dk.Name, systemUUID, "").
			Return(objectID, nil)

		r := NewReconciler()
		actual, err := r.createK8sConnectionSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, objectID, actual)
	})
	t.Run("creates setting - respect FF", func(t *testing.T) {
		const specialName = "I-AM-SPECIAL"
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)
		setClusterNameFF(dk, specialName)

		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, specialName, systemUUID, "").
			Return(objectID, nil)

		r := NewReconciler()
		actual, err := r.createK8sConnectionSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, objectID, actual)
	})

	t.Run("create kubernetes settings fails", func(t *testing.T) {
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, dk.Name, systemUUID, "").
			Return("", errors.New("boom"))

		r := NewReconciler()
		actual, err := r.createK8sConnectionSettingIfAbsent(t.Context(), dtClient, dk)

		require.Error(t, err)
		assert.Empty(t, actual)
	})
}

func TestRefreshMEIDWithRetry(t *testing.T) {
	const (
		meID       = "KUBERNETES_CLUSTER-119C75CCDA94799F"
		systemUUID = "2132143215"
		objectID   = "2141rfa3sjvnsk"
	)

	t.Run("no retry on success", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: dk.Name}
		setSystemUUID(dk, systemUUID)
		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(me, nil)

		r := NewReconciler()
		err := r.refreshMEIDWithRetry(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, me.ID, dk.Status.KubernetesClusterMEID)
		assert.Equal(t, me.Name, dk.Status.KubernetesClusterName)
	})

	t.Run("retry on missing", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: dk.Name}
		setSystemUUID(dk, systemUUID)
		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(settings.K8sClusterME{}, nil).Once()
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(me, nil).Once()

		r := NewReconciler()
		err := r.refreshMEIDWithRetry(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, me.ID, dk.Status.KubernetesClusterMEID)
		assert.Equal(t, me.Name, dk.Status.KubernetesClusterName)
	})

	t.Run("error after no success for 5 tries", func(t *testing.T) {
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)
		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(settings.K8sClusterME{}, nil).Times(5)

		r := NewReconciler()
		err := r.refreshMEIDWithRetry(t.Context(), dtClient, dk)
		require.Error(t, err)
	})

	t.Run("instant return on random error", func(t *testing.T) {
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)
		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(settings.K8sClusterME{}, errors.New("BOOM"))

		r := NewReconciler()
		err := r.refreshMEIDWithRetry(t.Context(), dtClient, dk)
		require.Error(t, err)
	})
}

func TestCreateK8sAppSettingIfAbsent(t *testing.T) {
	const (
		meID   = "KUBERNETES_CLUSTER-119C75CCDA94799F"
		meName = "my-cluster"
	)

	t.Run("don't create app without FF", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: meName}
		setMEInfo(dk, me)

		dtClient := settingsmock.NewAPIClient(t)

		r := NewReconciler()
		err := r.createK8sAppSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
	})

	t.Run("don't create app setting as settings already exist", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: meName}
		setMEInfo(dk, me)
		enableAppFF(dk)

		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{TotalCount: 1}, nil)

		r := NewReconciler()
		err := r.createK8sAppSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
	})

	t.Run("don't create app setting when get entities api response is error", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: meName}
		setMEInfo(dk, me)
		enableAppFF(dk)

		expectErr := errors.New("could not get monitored entities")

		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{}, expectErr)

		r := NewReconciler()
		err := r.createK8sAppSettingIfAbsent(t.Context(), dtClient, dk)
		require.ErrorIs(t, err, expectErr)
	})

	t.Run("don't create app setting when get CreateOrUpdateKubernetesAppSetting response is error", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: meName}
		setMEInfo(dk, me)
		enableAppFF(dk)

		expectErr := errors.New("could not create app setting")

		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{}, nil)
		dtClient.EXPECT().
			CreateOrUpdateKubernetesAppSetting(anyCtx, me.ID).
			Return("", expectErr)

		r := NewReconciler()
		err := r.createK8sAppSettingIfAbsent(t.Context(), dtClient, dk)
		require.ErrorIs(t, err, expectErr)
	})

	t.Run("create app setting as settings don't already exist", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: meName}
		setMEInfo(dk, me)
		enableAppFF(dk)

		r := NewReconciler()

		dtClient := settingsmock.NewAPIClient(t)
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{}, nil)
		dtClient.EXPECT().
			CreateOrUpdateKubernetesAppSetting(anyCtx, me.ID).
			Return("test", nil)

		err := r.createK8sAppSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
	})
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
