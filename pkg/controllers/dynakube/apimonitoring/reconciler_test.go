package apimonitoring

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testUID      = "test-uid"
	testName     = "test-clusterLabel"
	testObjectID = "test-objectid"
)

var mockCtx = mock.MatchedBy(func(context.Context) bool { return true })

func TestNewDefaultReconiler(t *testing.T) {
	createDefaultReconciler(t)
}

func createDefaultReconciler(t *testing.T) Reconciler {
	return createReconciler(t, dtclient.K8sClusterME{}, dtclient.GetSettingsResponse{TotalCount: 0}, "", "")
}

func createReconciler(t *testing.T, monitoredEntity dtclient.K8sClusterME, getSettingsResponse dtclient.GetSettingsResponse, objectID string, meID interface{}) Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetK8sClusterME", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return(monitoredEntity, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), dtclient.K8sClusterME{ID: "test-MEID"},
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), monitoredEntity,
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), monitoredEntity,
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, mock.AnythingOfType("string")).
		Return(objectID, nil)
	mockClient.On("CreateOrUpdateKubernetesAppSetting", mock.AnythingOfType("context.backgroundCtx"), meID).
		Return("transitionSchemaObjectID", nil)

	for _, call := range mockClient.ExpectedCalls {
		call.Maybe()
	}

	dk := newDynaKube()
	passMonitoredEntities := createPassingReconciler(t)
	r := Reconciler{
		dtc:                 mockClient,
		dk:                  dk,
		k8sEntityReconciler: passMonitoredEntities,
		clusterLabel:        testName,
	}

	return r
}

func createReadOnlyReconciler(t *testing.T, monitoredEntity dtclient.K8sClusterME, getSettingsResponse dtclient.GetSettingsResponse) Reconciler {
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetK8sClusterME", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return(monitoredEntity, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), dtclient.K8sClusterME{ID: "test-MEID"},
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), monitoredEntity,
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), monitoredEntity,
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, "KUBERNETES_CLUSTER-119C75CCDA94799F").
		Return("", errors.New("BOOM, readonly only client is used"))
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, "test-MEID").
		Return("", errors.New("BOOM, readonly only client is used"))
	mockClient.On("CreateOrUpdateKubernetesAppSetting", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return("", errors.New("BOOM, readonly only client is used"))

	for _, call := range mockClient.ExpectedCalls {
		call.Maybe()
	}

	dk := newDynaKube()
	passMonitoredEntities := createPassingReconciler(t)
	r := Reconciler{
		dtc:                 mockClient,
		dk:                  dk,
		k8sEntityReconciler: passMonitoredEntities,
		clusterLabel:        testName,
	}

	return r
}

func createReconcilerWithError(t *testing.T, monitoredEntitiesError error, getSettingsResponseError error, createSettingsResponseError error, createAppSettingsResponseError error) Reconciler {
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetK8sClusterME", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return(dtclient.K8sClusterME{}, monitoredEntitiesError)
	mockClient.On("GetSettingsForMonitoredEntity",
		mock.AnythingOfType("context.backgroundCtx"),
		mock.AnythingOfType("dynatrace.K8sClusterME"),
		mock.AnythingOfType("string")).
		Return(dtclient.GetSettingsResponse{}, getSettingsResponseError)
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, "KUBERNETES_CLUSTER-119C75CCDA94799F").
		Return("", createSettingsResponseError)
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, "test-MEID").
		Return("", createSettingsResponseError)
	mockClient.On("CreateOrUpdateKubernetesAppSetting", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return("", createAppSettingsResponseError)

	for _, call := range mockClient.ExpectedCalls {
		call.Maybe()
	}

	dk := newDynaKube()
	passMonitoredEntities := createPassingReconciler(t)
	r := Reconciler{
		dtc:                 mockClient,
		dk:                  dk,
		k8sEntityReconciler: passMonitoredEntities,
		clusterLabel:        testName,
	}

	return r
}

func createMonitoredEntities() dtclient.K8sClusterME {
	return dtclient.K8sClusterME{
		ID: "KUBERNETES_CLUSTER-119C75CCDA94799F", Name: "operator test entity 1",
	}
}

func TestReconcile(t *testing.T) {
	t.Run("reconciler does not fail in with defaults", func(t *testing.T) {
		r := createDefaultReconciler(t)

		err := r.Reconcile(context.Background())

		require.NoError(t, err)
	})

	t.Run("create setting when no monitored entities are existing", func(t *testing.T) {
		r := createReconciler(t, dtclient.K8sClusterME{}, dtclient.GetSettingsResponse{}, testObjectID, "")

		actual, err := r.createObjectIDIfNotExists(context.Background())

		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run("create setting when no settings for the found monitored entities are existing", func(t *testing.T) {
		entities := createMonitoredEntities()
		r := createReconciler(t, entities, dtclient.GetSettingsResponse{}, testObjectID, "")

		actual, err := r.createObjectIDIfNotExists(context.Background())

		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run("don't create setting when settings for the found monitored entities are existing", func(t *testing.T) {
		entities := createMonitoredEntities()
		r := createReadOnlyReconciler(t, entities, dtclient.GetSettingsResponse{TotalCount: 1})

		actual, err := r.createObjectIDIfNotExists(context.Background())

		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("optional scope settings.write not available", func(t *testing.T) {
		r := createReconcilerWithError(t, nil, errors.New("Unauthorized, missing token scopes"), nil, nil)
		conditions.SetOptionalScopeMissing(&r.dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsWrite, "not available")
		err := r.Reconcile(context.Background())

		require.NoError(t, err)
	})

	t.Run("app-transition schema not found", func(t *testing.T) {
		testME := dtclient.K8sClusterME{ID: "test-MEID"}

		mockClient := dtclientmock.NewClient(t)
		mockClient.EXPECT().
			GetSettingsForMonitoredEntity(mockCtx, testME, dtclient.KubernetesSettingsSchemaID).
			Return(dtclient.GetSettingsResponse{TotalCount: 1}, nil).Once()
		mockClient.EXPECT().
			GetSettingsForMonitoredEntity(mockCtx, testME, dtclient.AppTransitionSchemaID).
			Return(dtclient.GetSettingsResponse{}, dtclient.ServerError{Code: 404}).Once()

		dk := newDynaKube()
		passMonitoredEntities := createPassingReconciler(t)
		r := Reconciler{
			dtc:                 mockClient,
			dk:                  dk,
			k8sEntityReconciler: passMonitoredEntities,
			clusterLabel:        testName,
		}

		err := r.Reconcile(t.Context())
		require.NoError(t, err)
	})
}

func TestReconcileErrors(t *testing.T) {
	t.Run("don't create setting when no kube-system uuid is given", func(t *testing.T) {
		r := createReconciler(t, dtclient.K8sClusterME{ID: "test-MEID"}, dtclient.GetSettingsResponse{}, testObjectID, "")
		r.dk.Status.KubeSystemUUID = ""

		actual, err := r.createObjectIDIfNotExists(context.Background())

		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("don't create setting when get entities api response is error", func(t *testing.T) {
		r := createReconcilerWithError(t, nil, errors.New("could not get monitored entities"), nil, nil)

		actual, err := r.createObjectIDIfNotExists(context.Background())

		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("don't create setting when get settings api response is error", func(t *testing.T) {
		r := createReconcilerWithError(t, nil, errors.New("could not get settings for monitored entities"), nil, nil)

		actual, err := r.createObjectIDIfNotExists(context.Background())

		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("don't create setting when create settings api response is error", func(t *testing.T) {
		r := createReconcilerWithError(t, nil, nil, errors.New("could not create monitored entity"), nil)

		actual, err := r.createObjectIDIfNotExists(context.Background())

		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("create settings successful in case of CreateOrUpdateKubernetesAppSetting error", func(t *testing.T) {
		r := createReconcilerWithError(t, nil, nil, nil, errors.New("could not create monitored entity"))
		r.dk.Status.KubeSystemUUID = "test-uid"

		_, err := r.createObjectIDIfNotExists(context.Background())

		require.NoError(t, err)
	})
}

func TestHandleKubernetesAppEnabled(t *testing.T) {
	t.Run("don't create app setting due to empty MonitoredEntitys", func(t *testing.T) {
		r := createReconciler(t, dtclient.K8sClusterME{}, dtclient.GetSettingsResponse{}, "", "")

		err := r.handleKubernetesAppEnabled(context.Background(), dtclient.K8sClusterME{})
		require.NoError(t, err)
	})

	t.Run("don't create app setting as settings already exist", func(t *testing.T) {
		entities := createMonitoredEntities()
		r := createReconciler(t, entities, dtclient.GetSettingsResponse{TotalCount: 1}, "", "")

		err := r.handleKubernetesAppEnabled(context.Background(), entities)
		require.NoError(t, err)
	})

	t.Run("don't create app setting when get entities api response is error", func(t *testing.T) {
		r := createReconcilerWithError(t, nil, errors.New("could not get monitored entities"), nil, nil)

		err := r.handleKubernetesAppEnabled(context.Background(), dtclient.K8sClusterME{})
		require.Error(t, err)
	})

	t.Run("don't create app setting when get CreateOrUpdateKubernetesAppSetting response is error", func(t *testing.T) {
		r := createReconcilerWithError(t, nil, nil, nil, errors.New("could not get monitored entities"))
		meID := "KUBERNETES_CLUSTER-0E30FE4BF2007587"
		entity := dtclient.K8sClusterME{ID: meID, Name: "operator test entity newest"}

		err := r.handleKubernetesAppEnabled(context.Background(), entity)
		require.Error(t, err)
	})

	t.Run("create app setting as settings already exist", func(t *testing.T) {
		meID := "KUBERNETES_CLUSTER-0E30FE4BF2007587"
		entities := dtclient.K8sClusterME{
			ID: meID, Name: "operator test entity newest",
		}
		r := createReconciler(t, entities, dtclient.GetSettingsResponse{}, "", meID)
		err := r.handleKubernetesAppEnabled(context.Background(), entities)
		require.NoError(t, err)
	})
}

func newDynaKube() *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DynaKube",
			APIVersion: "dynatrace.com/v1beta1",
		},
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
			KubeSystemUUID:        "test-uid",
			KubernetesClusterMEID: "test-MEID",
		},
	}
	conditions.SetOptionalScopeAvailable(&dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsRead, "available")
	conditions.SetOptionalScopeAvailable(&dk.Status.Conditions, dtclient.ConditionTypeAPITokenSettingsWrite, "available")

	return dk
}

func createPassingReconciler(t *testing.T) *controllermock.Reconciler {
	passMock := controllermock.NewReconciler(t)
	passMock.On("Reconcile", mock.Anything).Return(nil).Maybe()

	return passMock
}
