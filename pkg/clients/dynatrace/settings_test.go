package dynatrace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUID      = "test-uid"
	testName     = "test-name"
	testObjectID = "test-objectid"
	testScope    = "test-scope"
)

func TestDynatraceClient_GetMonitoredEntitiesForKubeSystemUUID(t *testing.T) {
	ctx := context.Background()

	t.Run("monitored entities for this uuid exist", func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createEntitiesMockParams(expected, http.StatusOK)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetMonitoredEntitiesForKubeSystemUUID(ctx, testUID)

		// assert
		require.NotNil(t, actual)
		require.NoError(t, err)
		assert.Len(t, actual, 2)
		assert.Equal(t, expected, actual)
	})

	t.Run("no monitored entities for this uuid exist", func(t *testing.T) {
		// arrange
		expected := []MonitoredEntity{}

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createEntitiesMockParams(expected, http.StatusOK)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetMonitoredEntitiesForKubeSystemUUID(ctx, testUID)

		// assert
		require.NotNil(t, actual)
		require.NoError(t, err)
		assert.Empty(t, actual)
		assert.Equal(t, expected, actual)
	})

	t.Run("no monitored entities found because no kube-system uuid is provided", func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createEntitiesMockParams(expected, http.StatusBadRequest)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetMonitoredEntitiesForKubeSystemUUID(ctx, "")

		// assert
		require.Nil(t, actual)
		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("no monitored entities found because of an api error", func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createEntitiesMockParams(expected, http.StatusBadRequest)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetMonitoredEntitiesForKubeSystemUUID(ctx, testUID)

		// assert
		require.Nil(t, actual)
		require.Error(t, err)
		assert.Empty(t, actual)
	})
}

func TestDynatraceClient_GetSettingsForMonitoredEntity(t *testing.T) {
	ctx := context.Background()

	t.Run(`settings for the given monitored entities exist`, func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()
		totalCount := 2

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(totalCount, "", http.StatusOK)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntity(ctx, &expected[0], KubernetesSettingsSchemaId)

		// assert
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Positive(t, actual.TotalCount)
		assert.Len(t, expected, actual.TotalCount)
	})

	t.Run(`no settings for the given monitored entities exist`, func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()
		totalCount := 0

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(totalCount, "", http.StatusOK)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntity(ctx, &expected[0], KubernetesSettingsSchemaId)

		// assert
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Less(t, actual.TotalCount, 1)
	})

	t.Run(`no settings for an empty list of monitored entities exist`, func(t *testing.T) {
		// it is immaterial what we put here since no http call is executed when the list of
		// monitored entities is empty, therefore also no settings will be returned
		totalCount := 999

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(totalCount, "", http.StatusOK)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntity(ctx, nil, KubernetesSettingsSchemaId)

		// assert
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Empty(t, actual.TotalCount)
	})

	t.Run(`no settings found for because of an api error`, func(t *testing.T) {
		// arrange
		entities := createMonitoredEntitiesForTesting()
		// it is immaterial what we put here since the http request is producing an error
		totalCount := 999

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(totalCount, "", http.StatusBadRequest)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntity(ctx, &entities[0], KubernetesSettingsSchemaId)

		// assert
		require.Error(t, err)
		assert.Empty(t, actual.TotalCount)
	})
}

func createMonitoredEntitiesForTesting() []MonitoredEntity {
	return []MonitoredEntity{
		{EntityId: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity 1", LastSeenTms: 1639483869085},
		{EntityId: "KUBERNETES_CLUSTER-119C75CCDA94799F", DisplayName: "operator test entity 2", LastSeenTms: 1639034988126},
	}
}
