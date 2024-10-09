package apimonitoring

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
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

func createDefaultReconciler(t *testing.T) *Reconciler {
	return createReconciler(t, newDynaKube(), []dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{TotalCount: 0}, "", "")
}

func createReconciler(t *testing.T, dk *dynakube.DynaKube, monitoredEntities []dtclient.MonitoredEntity, getSettingsResponse dtclient.GetSettingsResponse, objectID string, meID interface{}) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return(monitoredEntities, nil)
	mockClient.On("GetSettingsForMonitoredEntities", mock.AnythingOfType("context.backgroundCtx"), []dtclient.MonitoredEntity{{EntityId: "test-MEID"}},
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntities", mock.AnythingOfType("context.backgroundCtx"), []dtclient.MonitoredEntity{{EntityId: "KUBERNETES_CLUSTER-119C75CCDA94799F"}},
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntities", mock.AnythingOfType("context.backgroundCtx"), monitoredEntities,
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, mock.AnythingOfType("string")).
		Return(objectID, nil)
	mockClient.On("CreateOrUpdateKubernetesAppSetting", mock.AnythingOfType("context.backgroundCtx"), meID).
		Return("transitionSchemaObjectID", nil)

	for _, call := range mockClient.ExpectedCalls {
		call.Maybe()
	}

	r := NewReconciler(mockClient, dk, testName)
	require.NotNil(t, r)
	require.NotNil(t, r.dtc)

	return r
}

func createReadOnlyReconciler(t *testing.T, dk *dynakube.DynaKube, monitoredEntities []dtclient.MonitoredEntity, getSettingsResponse dtclient.GetSettingsResponse) *Reconciler {
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return(monitoredEntities, nil)
	mockClient.On("GetSettingsForMonitoredEntities", mock.AnythingOfType("context.backgroundCtx"), []dtclient.MonitoredEntity{{EntityId: "test-MEID"}},
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntities", mock.AnythingOfType("context.backgroundCtx"), []dtclient.MonitoredEntity{{EntityId: "KUBERNETES_CLUSTER-119C75CCDA94799F"}},
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("GetSettingsForMonitoredEntities", mock.AnythingOfType("context.backgroundCtx"), monitoredEntities,
		mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, "KUBERNETES_CLUSTER-119C75CCDA94799F").
		Return("", fmt.Errorf("BOOM, readonly only client is used"))
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, "test-MEID").
		Return("", fmt.Errorf("BOOM, readonly only client is used"))
	mockClient.On("CreateOrUpdateKubernetesAppSetting", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return("", fmt.Errorf("BOOM, readonly only client is used"))

	for _, call := range mockClient.ExpectedCalls {
		call.Maybe()
	}

	r := NewReconciler(mockClient, dk, testName)
	require.NotNil(t, r)
	require.NotNil(t, r.dtc)

	return r
}

func createReconcilerWithError(t *testing.T, dk *dynakube.DynaKube, monitoredEntitiesError error, getSettingsResponseError error, createSettingsResponseError error, createAppSettingsResponseError error) *Reconciler { //nolint:revive
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return([]dtclient.MonitoredEntity{}, monitoredEntitiesError)
	mockClient.On("GetSettingsForMonitoredEntities",
		mock.AnythingOfType("context.backgroundCtx"),
		mock.AnythingOfType("[]dynatrace.MonitoredEntity"),
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

	r := NewReconciler(mockClient, dk, testName)
	require.NotNil(t, r)
	require.NotNil(t, r.dtc)

	return r
}

func createMonitoredEntities() []dtclient.MonitoredEntity {
	return []dtclient.MonitoredEntity{
		{EntityId: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity 1", LastSeenTms: 1639483869085},
		{EntityId: "KUBERNETES_CLUSTER-119C75CCDA94799F", DisplayName: "operator test entity 2", LastSeenTms: 1639034988126},
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
		r := createReconciler(t, dk, []dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{}, testObjectID, "")

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run("create setting when no settings for the found monitored entities are existing", func(t *testing.T) {
		// arrange
		entities := createMonitoredEntities()
		r := createReconciler(t, dk, entities, dtclient.GetSettingsResponse{}, testObjectID, "")

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run("don't create setting when settings for the found monitored entities are existing", func(t *testing.T) {
		// arrange
		entities := createMonitoredEntities()
		r := createReadOnlyReconciler(t, dk, entities, dtclient.GetSettingsResponse{TotalCount: 1})

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.NoError(t, err)
		assert.Equal(t, "", actual)
	})
}

func TestReconcileErrors(t *testing.T) {
	ctx := context.Background()
	dk := newDynaKube()

	t.Run("don't create setting when no kube-system uuid is given", func(t *testing.T) {
		// arrange
		r := createReconciler(t, dk, []dtclient.MonitoredEntity{{EntityId: "test-MEID"}}, dtclient.GetSettingsResponse{}, testObjectID, "")
		dk.Status.KubeSystemUUID = ""

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run("don't create setting when get entities api response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, errors.New("could not get monitored entities"), nil, nil, nil)

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run("don't create setting when get settings api response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, errors.New("could not get settings for monitored entities"), nil, nil)

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run("don't create setting when create settings api response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, nil, errors.New("could not create monitored entity"), nil)

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run("create settings successful in case of CreateOrUpdateKubernetesAppSetting error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, nil, nil, errors.New("could not create monitored entity"))
		dk.Status.KubeSystemUUID = "test-uid"

		// act
		_, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.NoError(t, err)
	})
}

func TestHandleKubernetesAppEnabled(t *testing.T) {
	ctx := context.Background()
	dk := newDynaKube()

	t.Run("don't create app setting due to empty MonitoredEntitys", func(t *testing.T) {
		// arrange
		r := createReconciler(t, dk, []dtclient.MonitoredEntity{{EntityId: "test-MEID"}},
			dtclient.GetSettingsResponse{}, "", "")

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, []dtclient.MonitoredEntity{{EntityId: "test-MEID"}})

		// assert
		require.NoError(t, err)
	})

	t.Run("don't create app setting as settings already exist", func(t *testing.T) {
		// arrange
		entities := []dtclient.MonitoredEntity{
			{EntityId: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity newest", LastSeenTms: 1639483869085},
			{EntityId: "KUBERNETES_CLUSTER-119C75CCDA94799F", DisplayName: "operator test entity 1", LastSeenTms: 1639034988126},
		}
		r := createReconciler(t, dk, entities, dtclient.GetSettingsResponse{TotalCount: 1}, "", "")

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, entities)

		// assert
		require.NoError(t, err)
	})

	t.Run("don't create app setting when get entities api response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, errors.New("could not get monitored entities"), nil, nil)

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, []dtclient.MonitoredEntity{})

		// assert
		require.Error(t, err)
	})

	t.Run("don't create app setting when get CreateOrUpdateKubernetesAppSetting response is error", func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dk, nil, nil, nil, errors.New("could not get monitored entities"))
		meID := "KUBERNETES_CLUSTER-0E30FE4BF2007587"
		entities := []dtclient.MonitoredEntity{
			{EntityId: meID, DisplayName: "operator test entity newest", LastSeenTms: 1639483869085},
		}
		// act
		_, err := r.handleKubernetesAppEnabled(ctx, entities)

		// assert
		require.Error(t, err)
	})

	t.Run("create app setting as settings already exist", func(t *testing.T) {
		// arrange
		meID := "KUBERNETES_CLUSTER-0E30FE4BF2007587"
		entities := []dtclient.MonitoredEntity{
			{EntityId: meID, DisplayName: "operator test entity newest", LastSeenTms: 1639483869085},
		}
		r := createReconciler(t, dk, entities, dtclient.GetSettingsResponse{}, "", meID)
		// act
		id, err := r.handleKubernetesAppEnabled(ctx, entities)
		// assert
		require.NoError(t, err)
		assert.Equal(t, "transitionSchemaObjectID", id)
	})
}

func TestDetermineNewestMonitoredEntity(t *testing.T) {
	t.Run("newest monitored entity is correctly calculated", func(t *testing.T) {
		// arrange
		// explicit create of entities here to visualize that one has the newest LastSeenTimestamp
		// here it is the first one
		entities := []dtclient.MonitoredEntity{
			{EntityId: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity newest", LastSeenTms: 1639483869085},
			{EntityId: "KUBERNETES_CLUSTER-119C75CCDA94799F", DisplayName: "operator test entity 1", LastSeenTms: 1639034988126},
			{EntityId: "KUBERNETES_CLUSTER-119C75CCDA947993", DisplayName: "operator test entity 2", LastSeenTms: 1639134988126},
			{EntityId: "KUBERNETES_CLUSTER-119C75CCDA94799D", DisplayName: "operator test entity 3", LastSeenTms: 1639234988126},
		}

		// act
		newestEntity := determineNewestMonitoredEntity(entities)

		// assert
		assert.NotNil(t, newestEntity)
		assert.Equal(t, entities[0].EntityId, newestEntity)
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
				dynakube.AnnotationFeatureK8sAppEnabled: "true",
			},
		},
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				HostMonitoring: &dynakube.HostInjectSpec{},
			},
		},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID:        "test-uid",
			KubernetesClusterMEID: "test-MEID",
		},
	}
}
