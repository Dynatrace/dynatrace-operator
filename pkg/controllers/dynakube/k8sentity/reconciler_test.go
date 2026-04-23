package k8sentity

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/tenant/optionalscopes"
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

	optionalscopes.Available(dk.OptionalScopes(), token.ScopeSettingsRead)
	optionalscopes.Available(dk.OptionalScopes(), token.ScopeSettingsWrite)

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
		optionalscopes.Missing(dk.OptionalScopes(), token.ScopeSettingsRead)

		r := NewReconciler()
		err := r.Reconcile(t.Context(), settingsmock.NewClient(t), dk)
		require.NoError(t, err)
	})

	t.Run("missing kube-system uuid", func(t *testing.T) {
		dk := newDynaKube()

		r := NewReconciler()
		err := r.Reconcile(t.Context(), settingsmock.NewClient(t), dk)

		require.ErrorIs(t, err, errMissingKubeSystemUUID)
	})

	t.Run("refreshes MEID immediately after creating settings on first run", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: dk.Name}
		enableKubeMon(dk)
		enableAppFF(dk)
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewClient(t)
		// 1. reconcileClusterMEID: no ME found yet (settings object not yet created)
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(settings.K8sClusterME{}, nil).Once()
		// 2. createObjectIDIfNotExists: no settings exist yet, create them
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, me.Name, systemUUID, "").
			Return(me.ID, nil).Once()
		// 3. refreshClusterMEID: ME is now available after settings were created
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(me, nil).Once()
		// 4. reconcile app transition schema
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{}, nil).Once()
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

		dtClient := settingsmock.NewClient(t)
		// 1. reconcileClusterMEID: no ME found yet (settings object not yet created)
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(me, nil).Once()
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

		dtClient := settingsmock.NewClient(t)
		// 1. reconcileClusterMEID: no ME found yet (settings object not yet created)
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(settings.K8sClusterME{}, nil).Once()
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, me.Name, systemUUID, "").
			Return(me.ID, nil).Once()
		// 3. refreshClusterMEID: ME is now available after settings were created
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(me, nil).Once()

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

		dtClient := settingsmock.NewClient(t)
		r := NewReconciler()
		err := r.reconcileMEID(t.Context(), dtClient, dk)
		require.NoError(t, err)
	})

	t.Run("sets MEID when ME is found", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: meName}
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().
			GetK8sClusterME(anyCtx, systemUUID).
			Return(me, nil).Once()

		r := NewReconciler()
		err := r.reconcileMEID(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, me.ID, dk.Status.KubernetesClusterMEID)
		assert.Equal(t, me.Name, dk.Status.KubernetesClusterName)
	})

	t.Run("no error if no MEs are found", func(t *testing.T) {
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)
		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(settings.K8sClusterME{}, nil).Once()

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

		dtClient := settingsmock.NewClient(t)

		r := NewReconciler()
		actual, err := r.createK8sConnectionSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Empty(t, actual)
	})
	t.Run("creates setting", func(t *testing.T) {
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, dk.Name, systemUUID, "").
			Return(objectID, nil).Once()

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

		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, specialName, systemUUID, "").
			Return(objectID, nil).Once()

		r := NewReconciler()
		actual, err := r.createK8sConnectionSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
		assert.Equal(t, objectID, actual)
	})

	t.Run("create kubernetes settings fails", func(t *testing.T) {
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)

		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().
			CreateOrUpdateKubernetesSetting(anyCtx, dk.Name, systemUUID, "").
			Return("", errors.New("boom")).Once()

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
		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(me, nil).Once()

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
		dtClient := settingsmock.NewClient(t)
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
		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(settings.K8sClusterME{}, nil).Times(5)

		r := NewReconciler()
		err := r.refreshMEIDWithRetry(t.Context(), dtClient, dk)
		require.Error(t, err)
	})

	t.Run("instant return on random error", func(t *testing.T) {
		dk := newDynaKube()
		setSystemUUID(dk, systemUUID)
		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().GetK8sClusterME(anyCtx, systemUUID).Return(settings.K8sClusterME{}, errors.New("BOOM")).Once()

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

		dtClient := settingsmock.NewClient(t)

		r := NewReconciler()
		err := r.createK8sAppSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
	})

	t.Run("don't create app setting as settings already exist", func(t *testing.T) {
		dk := newDynaKube()
		me := settings.K8sClusterME{ID: meID, Name: meName}
		setMEInfo(dk, me)
		enableAppFF(dk)

		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{TotalCount: 1}, nil).Once()

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

		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{}, expectErr).Once()

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

		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{}, nil).Once()
		dtClient.EXPECT().
			CreateOrUpdateKubernetesAppSetting(anyCtx, me.ID).
			Return("", expectErr).Once()

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

		dtClient := settingsmock.NewClient(t)
		dtClient.EXPECT().
			GetSettingsForMonitoredEntity(anyCtx, me, settings.AppTransitionSchemaID).
			Return(settings.TotalCountSettingsResponse{}, nil).Once()
		dtClient.EXPECT().
			CreateOrUpdateKubernetesAppSetting(anyCtx, me.ID).
			Return("test", nil).Once()

		err := r.createK8sAppSettingIfAbsent(t.Context(), dtClient, dk)
		require.NoError(t, err)
	})
}
