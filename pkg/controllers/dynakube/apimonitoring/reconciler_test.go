package apimonitoring

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
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
	return createReconciler(t, newDynaKube(), testUID, []dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{TotalCount: 0}, "", "")
}

func createReconciler(t *testing.T, dynakube *dynatracev1beta1.DynaKube, uid string, monitoredEntities []dtclient.MonitoredEntity, getSettingsResponse dtclient.GetSettingsResponse, objectID string, meID interface{}) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return(monitoredEntities, nil)
	mockClient.On("GetSettingsForMonitoredEntities", mock.AnythingOfType("context.backgroundCtx"), monitoredEntities, mock.AnythingOfType("string")).
		Return(getSettingsResponse, nil)
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, mock.AnythingOfType("string")).
		Return(objectID, nil)
	mockClient.On("CreateOrUpdateKubernetesAppSetting", mock.AnythingOfType("context.backgroundCtx"), meID).
		Return("transitionSchemaObjectID", nil)

	for _, call := range mockClient.ExpectedCalls {
		call.Maybe()
	}

	r := NewReconciler(mockClient, dynakube, testName, uid)
	require.NotNil(t, r)
	require.NotNil(t, r.dtc)

	return r
}

func createReconcilerWithError(t *testing.T, dynakube *dynatracev1beta1.DynaKube, monitoredEntitiesError error, getSettingsResponseError error, createSettingsResponseError error, createAppSettingsResponseError error) *Reconciler { //nolint:revive
	mockClient := dtclientmock.NewClient(t)
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return([]dtclient.MonitoredEntity{}, monitoredEntitiesError)
	mockClient.On("GetSettingsForMonitoredEntities",
		mock.AnythingOfType("context.backgroundCtx"),
		mock.AnythingOfType("[]dynatrace.MonitoredEntity"),
		mock.AnythingOfType("string")).
		Return(dtclient.GetSettingsResponse{}, getSettingsResponseError)
	mockClient.On("CreateOrUpdateKubernetesSetting", mock.AnythingOfType("context.backgroundCtx"), testName, testUID, mock.AnythingOfType("string")).
		Return("", createSettingsResponseError)
	mockClient.On("CreateOrUpdateKubernetesAppSetting", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).
		Return("", createAppSettingsResponseError)

	for _, call := range mockClient.ExpectedCalls {
		call.Maybe()
	}

	r := NewReconciler(mockClient, dynakube, testName, testUID)
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
	dynakube := newDynaKube()

	t.Run(`reconciler does not fail in with defaults`, func(t *testing.T) {
		// arrange
		r := createDefaultReconciler(t)

		// act
		err := r.Reconcile(ctx)

		// assert
		require.NoError(t, err)
	})

	t.Run(`create setting when no monitored entities are existing`, func(t *testing.T) {
		// arrange
		r := createReconciler(t, dynakube, testUID, []dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{}, testObjectID, "")

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run(`create setting when no settings for the found monitored entities are existing`, func(t *testing.T) {
		// arrange
		entities := createMonitoredEntities()
		r := createReconciler(t, dynakube, testUID, entities, dtclient.GetSettingsResponse{}, testObjectID, "")

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run(`don't create setting when settings for the found monitored entities are existing`, func(t *testing.T) {
		// arrange
		entities := createMonitoredEntities()
		r := createReconciler(t, dynakube, testUID, entities, dtclient.GetSettingsResponse{TotalCount: 1}, testObjectID, "")

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.NoError(t, err)
		assert.Equal(t, "", actual)
	})
}

func TestReconcileErrors(t *testing.T) {
	ctx := context.Background()
	dynakube := newDynaKube()

	t.Run(`don't create setting when no kube-system uuid is given`, func(t *testing.T) {
		// arrange
		r := createReconciler(t, dynakube, "", []dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{}, testObjectID, "")

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run(`don't create setting when get entities api response is error`, func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dynakube, errors.New("could not get monitored entities"), nil, nil, nil)

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run(`don't create setting when get settings api response is error`, func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dynakube, nil, errors.New("could not get settings for monitored entities"), nil, nil)

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run(`don't create setting when create settings api response is error`, func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dynakube, nil, nil, errors.New("could not create monitored entity"), nil)

		// act
		actual, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run(`create settings successful in case of CreateOrUpdateKubernetesAppSetting error`, func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dynakube, nil, nil, nil, errors.New("could not create monitored entity"))

		// act
		_, err := r.createObjectIdIfNotExists(ctx)

		// assert
		require.NoError(t, err)
	})
}

func TestHandleKubernetesAppEnabled(t *testing.T) {
	ctx := context.Background()
	dynakube := newDynaKube()

	t.Run(`don't create app setting due to empty MonitoredEntitys`, func(t *testing.T) {
		// arrange
		r := createReconciler(t, dynakube, "", []dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{}, "", "")

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, []dtclient.MonitoredEntity{})

		// assert
		require.NoError(t, err)
	})

	t.Run(`don't create app setting as settings already exist`, func(t *testing.T) {
		// arrange
		entities := []dtclient.MonitoredEntity{
			{EntityId: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity newest", LastSeenTms: 1639483869085},
			{EntityId: "KUBERNETES_CLUSTER-119C75CCDA94799F", DisplayName: "operator test entity 1", LastSeenTms: 1639034988126},
		}
		r := createReconciler(t, dynakube, "", entities, dtclient.GetSettingsResponse{TotalCount: 1}, "", "")

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, entities)

		// assert
		require.NoError(t, err)
	})

	t.Run(`don't create app setting when get entities api response is error`, func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dynakube, nil, errors.New("could not get monitored entities"), nil, nil)

		// act
		_, err := r.handleKubernetesAppEnabled(ctx, []dtclient.MonitoredEntity{})

		// assert
		require.Error(t, err)
	})

	t.Run(`don't create app setting when get CreateOrUpdateKubernetesAppSetting response is error`, func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, dynakube, nil, nil, nil, errors.New("could not get monitored entities"))
		meID := "KUBERNETES_CLUSTER-0E30FE4BF2007587"
		entities := []dtclient.MonitoredEntity{
			{EntityId: meID, DisplayName: "operator test entity newest", LastSeenTms: 1639483869085},
		}
		// act
		_, err := r.handleKubernetesAppEnabled(ctx, entities)

		// assert
		require.Error(t, err)
	})

	t.Run(`create app setting as settings already exist`, func(t *testing.T) {
		// arrange
		meID := "KUBERNETES_CLUSTER-0E30FE4BF2007587"
		entities := []dtclient.MonitoredEntity{
			{EntityId: meID, DisplayName: "operator test entity newest", LastSeenTms: 1639483869085},
		}
		r := createReconciler(t, dynakube, "", entities, dtclient.GetSettingsResponse{}, "", meID)
		// act
		id, err := r.handleKubernetesAppEnabled(ctx, entities)
		// assert
		require.NoError(t, err)
		assert.Equal(t, "transitionSchemaObjectID", id)
	})
}

func TestDetermineNewestMonitoredEntity(t *testing.T) {
	t.Run(`newest monitored entity is correctly calculated`, func(t *testing.T) {
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

func newDynaKube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DynaKube",
			APIVersion: "dynatrace.com/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-oneagent",
			Namespace: "my-namespace",
			UID:       "69e98f18-805a-42de-84b5-3eae66534f75",
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureK8sAppEnabled: "true",
			},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}
}
