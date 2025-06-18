package apimonitoring

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
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

func TestNewDefaultReconiler(t *testing.T) {
	createDefaultReconciler(t)
}

func createDefaultReconciler(t *testing.T) Reconciler {
	return createReconciler(t, newDynaKube(), []dtclient.MonitoredEntity{}, nil, dtclient.GetSettingsResponse{TotalCount: 0}, "", "")
}

func createReconciler(t *testing.T, dk *dynakube.DynaKube, monitoredEntities []dtclient.MonitoredEntity, monitoredEntity *dtclient.MonitoredEntity, getSettingsResponse dtclient.GetSettingsResponse, objectID string, meID interface{}) Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return(monitoredEntities, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), &dtclient.MonitoredEntity{EntityID: "test-MEID"},
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), &dtclient.MonitoredEntity{EntityID: "KUBERNETES_CLUSTER-119C75CCDA94799F"},
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

	passMonitoredEntities := createPassingReconciler(t)
	r := Reconciler{
		dtc:                         mockClient,
		dk:                          dk,
		monitoredEntitiesReconciler: passMonitoredEntities,
		clusterLabel:                testName,
	}

	return r
}

func createReadOnlyReconciler(t *testing.T, dk *dynakube.DynaKube, monitoredEntities []dtclient.MonitoredEntity, monitoredEntity *dtclient.MonitoredEntity, getSettingsResponse dtclient.GetSettingsResponse) Reconciler {
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return(monitoredEntities, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), &dtclient.MonitoredEntity{EntityID: "test-MEID"},
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntity", mock.AnythingOfType("context.backgroundCtx"), &dtclient.MonitoredEntity{EntityID: "KUBERNETES_CLUSTER-119C75CCDA94799F"},
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

	passMonitoredEntities := createPassingReconciler(t)
	r := Reconciler{
		dtc:                         mockClient,
		dk:                          dk,
		monitoredEntitiesReconciler: passMonitoredEntities,
		clusterLabel:                testName,
	}

	return r
}

func createReconcilerWithError(t *testing.T, dk *dynakube.DynaKube, monitoredEntitiesError error, getSettingsResponseError error, createSettingsResponseError error, createAppSettingsResponseError error) Reconciler { //nolint:revive
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return([]dtclient.MonitoredEntity{}, monitoredEntitiesError)
	mockClient.On("GetSettingsForMonitoredEntity",
		mock.AnythingOfType("context.backgroundCtx"),
		mock.AnythingOfType("*dynatrace.MonitoredEntity"),
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

	passMonitoredEntities := createPassingReconciler(t)
	r := Reconciler{
		dtc:                         mockClient,
		dk:                          dk,
		monitoredEntitiesReconciler: passMonitoredEntities,
		clusterLabel:                testName,
	}

	return r
}

func createMonitoredEntities() []dtclient.MonitoredEntity {
	return []dtclient.MonitoredEntity{
		{EntityID: "KUBERNETES_CLUSTER-119C75CCDA94799F", DisplayName: "operator test entity 1", LastSeenTms: 1639483869085},
		{EntityID: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity 2", LastSeenTms: 1639034988126},
	}
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	dk := newDynaKube()

	t.Run("reconciler does not fail in with defaults", func(t *testing.T) {
		// arrange
		r := createDefaultReconciler(t)

		// act
		err := r.Reconcile(ctx)

		// assert
		require.NoError(t, err)
	})

	t.Run("create setting when no monitored entities are existing", func(t *testing.T) {
		// arrange
		r := createReconciler(t, dk, []dtclient.MonitoredEntity{}, nil, dtclient.GetSettingsResponse{}, testObjectID, "")

		// act
		actual, err := r.createObjectIDIfNotExists(ctx)

		// assert
		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run("create setting when no settings for the found monitored entities are existing", func(t *testing.T) {
		// arrange
		entities := createMonitoredEntities()
		r := createReconciler(t, dk, entities, &entities[0], dtclient.GetSettingsResponse{}, testObjectID, "")

		// act
		actual, err := r.createObjectIDIfNotExists(ctx)

		// assert
		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run("don't create setting when settings for the found monitored entities are existing", func(t *testing.T) {
		// arrange
		entities := createMonitoredEntities()
		r := createReadOnlyReconciler(t, dk, entities, &entities[0], dtclient.GetSettingsResponse{TotalCount: 1})

		// act
		actual, err := r.createObjectIDIfNotExists(ctx)

		// assert
		require.NoError(t, err)
		assert.Empty(t, actual)
	})
}

func TestReconcileErrors(t *testing.T) {
	ctx := context.Background()
	dk := newDynaKube()

	t.Run("don't create setting when no kube-system uuid is given", func(t *testing.T) {
		// arrange
		r := createReconciler(t, dk, []dtclient.MonitoredEntity{{EntityID: "test-MEID"}}, &dtclient.MonitoredEntity{EntityID: "test-MEID"}, dtclient.GetSettingsResponse{}, testObjectID, "")
		dk.Status.KubeSystemUUID = ""

		// act
		actual, err := r.createObjectIDIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("don't create setting when get entities api response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, errors.New("could not get monitored entities"), nil, nil, nil)

		// act
		actual, err := r.createObjectIDIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("don't create setting when get settings api response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, errors.New("could not get settings for monitored entities"), nil, nil)

		// act
		actual, err := r.createObjectIDIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("don't create setting when create settings api response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, nil, errors.New("could not create monitored entity"), nil)

		// act
		actual, err := r.createObjectIDIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("create settings successful in case of CreateOrUpdateKubernetesAppSetting error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, nil, nil, errors.New("could not create monitored entity"))
		dk.Status.KubeSystemUUID = "test-uid"

		// act
		_, err := r.createObjectIDIfNotExists(ctx)

		// assert
		require.NoError(t, err)
	})
}

func TestHandleKubernetesAppEnabled(t *testing.T) {
	ctx := context.Background()
	dk := newDynaKube()

	t.Run("don't create app setting due to empty MonitoredEntitys", func(t *testing.T) {
		// arrange
		r := createReconciler(t, dk, []dtclient.MonitoredEntity{}, &dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{}, "", "")

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, &dtclient.MonitoredEntity{})

		// assert
		require.NoError(t, err)
	})

	t.Run("don't create app setting as settings already exist", func(t *testing.T) {
		// arrange
		entities := createMonitoredEntities()
		r := createReconciler(t, dk, entities, &entities[0], dtclient.GetSettingsResponse{TotalCount: 1}, "", "")

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, &entities[0])

		// assert
		require.NoError(t, err)
	})

	t.Run("don't create app setting when get entities api response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, errors.New("could not get monitored entities"), nil, nil)

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, &dtclient.MonitoredEntity{})

		// assert
		require.Error(t, err)
	})

	t.Run("don't create app setting when get CreateOrUpdateKubernetesAppSetting response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, nil, nil, errors.New("could not get monitored entities"))
		meID := "KUBERNETES_CLUSTER-0E30FE4BF2007587"
		entity := dtclient.MonitoredEntity{EntityID: meID, DisplayName: "operator test entity newest", LastSeenTms: 1639483869085}

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, &entity)

		// assert
		require.Error(t, err)
	})

	t.Run("create app setting as settings already exist", func(t *testing.T) {
		// arrange
		meID := "KUBERNETES_CLUSTER-0E30FE4BF2007587"
		entities := []dtclient.MonitoredEntity{
			{EntityID: meID, DisplayName: "operator test entity newest", LastSeenTms: 1639483869085},
		}
		r := createReconciler(t, dk, entities, &entities[0], dtclient.GetSettingsResponse{}, "", meID)
		// act
		id, err := r.handleKubernetesAppEnabled(ctx, &entities[0])
		// assert
		require.NoError(t, err)
		assert.Equal(t, "transitionSchemaObjectID", id)
	})
}

func newDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
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
}

func createPassingReconciler(t *testing.T) *controllermock.Reconciler {
	passMock := controllermock.NewReconciler(t)
	passMock.On("Reconcile", mock.Anything).Return(nil).Maybe()

	return passMock
}
