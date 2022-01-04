package automaticapimonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testUID      = "test-uid"
	testName     = "test-name"
	testObjectID = "test-objectid"
)

func TestNewDefaultReconiler(t *testing.T) {
	createDefaultReconciler(t)
}

func createDefaultReconciler(t *testing.T) *AutomaticApiMonitoringReconciler {
	return createReconciler(t, testUID, []dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{TotalCount: 0}, "")
}

func createReconciler(t *testing.T, uid string, monitoredEntities []dtclient.MonitoredEntity, getSettingsResponse dtclient.GetSettingsResponse, objectID string) *AutomaticApiMonitoringReconciler {
	mockClient := &dtclient.MockDynatraceClient{}
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("string")).
		Return(monitoredEntities, nil)
	mockClient.On("GetSettingsForMonitoredEntities", monitoredEntities).
		Return(getSettingsResponse, nil)
	mockClient.On("CreateKubernetesSetting", testName, testUID, mock.AnythingOfType("string")).
		Return(objectID, nil)

	r := NewReconciler(mockClient, testName, uid)
	require.NotNil(t, r)
	require.NotNil(t, r.dtc)

	return r
}

func createReconcilerWithError(t *testing.T, monitoredEntitiesError error, getSettingsResponseError error, createSettingsResponseError error) *AutomaticApiMonitoringReconciler {
	mockClient := &dtclient.MockDynatraceClient{}
	mockClient.On("GetMonitoredEntitiesForKubeSystemUUID", mock.AnythingOfType("string")).
		Return([]dtclient.MonitoredEntity{}, monitoredEntitiesError)
	mockClient.On("GetSettingsForMonitoredEntities", []dtclient.MonitoredEntity{}).
		Return(dtclient.GetSettingsResponse{}, getSettingsResponseError)
	mockClient.On("CreateKubernetesSetting", testName, testUID, mock.AnythingOfType("string")).
		Return("", createSettingsResponseError)

	r := NewReconciler(mockClient, testName, testUID)
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
	t.Run(`reconciler does not fail in with defaults`, func(t *testing.T) {
		// arrange
		r := createDefaultReconciler(t)

		// act
		err := r.Reconcile()

		// assert
		assert.NoError(t, err)
	})

	t.Run(`create setting when no monitored entities are existing`, func(t *testing.T) {
		// arrange
		r := createReconciler(t, testUID, []dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{}, testObjectID)

		// act
		actual, err := r.ensureSettingExists()

		// assert
		assert.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run(`create setting when no settings for the found monitored entities are existing`, func(t *testing.T) {
		// arrange
		entities := createMonitoredEntities()
		r := createReconciler(t, testUID, entities, dtclient.GetSettingsResponse{}, testObjectID)

		// act
		actual, err := r.ensureSettingExists()

		// assert
		assert.NoError(t, err)
		assert.Equal(t, testObjectID, actual)
	})

	t.Run(`don't create setting when settings for the found monitored entities are existing`, func(t *testing.T) {
		// arrange
		entities := createMonitoredEntities()
		r := createReconciler(t, testUID, entities, dtclient.GetSettingsResponse{TotalCount: 1}, testObjectID)

		// act
		actual, err := r.ensureSettingExists()

		// assert
		assert.NoError(t, err)
		assert.Equal(t, "", actual)
	})
}

func TestReconcileErrors(t *testing.T) {
	t.Run(`don't create setting when no kube-system uuid is given`, func(t *testing.T) {
		// arrange
		r := createReconciler(t, "", []dtclient.MonitoredEntity{}, dtclient.GetSettingsResponse{}, testObjectID)

		// act
		actual, err := r.ensureSettingExists()

		// assert
		assert.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run(`don't create setting when get entities api response is error`, func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, errors.New("could not get monitored entities"), nil, nil)

		// act
		actual, err := r.ensureSettingExists()

		// assert
		assert.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run(`don't create setting when get settings api response is error`, func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, nil, errors.New("could not get settings for monitored entities"), nil)

		// act
		actual, err := r.ensureSettingExists()

		// assert
		assert.Error(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run(`don't create setting when create settings api response is error`, func(t *testing.T) {
		// arrange
		r := createReconcilerWithError(t, nil, nil, errors.New("could not create monitored entity"))

		// act
		actual, err := r.ensureSettingExists()

		// assert
		assert.Error(t, err)
		assert.Equal(t, "", actual)
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
