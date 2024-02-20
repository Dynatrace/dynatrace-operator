package dynatrace

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

		dynatraceServer := httptest.NewServer(mockDynatraceServerEntitiesHandler(expected, false))
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
		assert.EqualValues(t, expected, actual)
	})

	t.Run("no monitored entities for this uuid exist", func(t *testing.T) {
		// arrange
		expected := []MonitoredEntity{}

		dynatraceServer := httptest.NewServer(mockDynatraceServerEntitiesHandler(expected, false))
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
		assert.EqualValues(t, expected, actual)
	})

	t.Run("no monitored entities found because no kube-system uuid is provided", func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()

		dynatraceServer := httptest.NewServer(mockDynatraceServerEntitiesHandler(expected, true))
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

		dynatraceServer := httptest.NewServer(mockDynatraceServerEntitiesHandler(expected, true))
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

func TestDynatraceClient_GetSettingsForMonitoredEntities(t *testing.T) {
	ctx := context.Background()

	t.Run(`settings for the given monitored entities exist`, func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()
		totalCount := 2

		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(totalCount, "", http.StatusOK))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntities(ctx, expected, SettingsSchemaId)

		// assert
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Greater(t, actual.TotalCount, 0)
		assert.Len(t, expected, actual.TotalCount)
	})

	t.Run(`no settings for the given monitored entities exist`, func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()
		totalCount := 0

		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(totalCount, "", http.StatusOK))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntities(ctx, expected, SettingsSchemaId)

		// assert
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Less(t, actual.TotalCount, 1)
	})

	t.Run(`no settings for an empty list of monitored entities exist`, func(t *testing.T) {
		// arrange
		entities := []MonitoredEntity{}
		// it is immaterial what we put here since no http call is executed when the list of
		// monitored entities is empty, therefore also no settings will be returned
		totalCount := 999

		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(totalCount, "", http.StatusOK))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntities(ctx, entities, SettingsSchemaId)

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

		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(totalCount, "", http.StatusBadRequest))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntities(ctx, entities, SettingsSchemaId)

		// assert
		require.Error(t, err)
		assert.Empty(t, actual.TotalCount)
	})
}

func TestDynatraceClient_CreateOrUpdateKubernetesSetting(t *testing.T) {
	ctx := context.Background()

	t.Run(`create settings with monitoring for the given monitored entity id`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, http.StatusOK))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.CreateOrUpdateKubernetesSetting(ctx, testName, testUID, testScope)

		// assert
		require.NotNil(t, actual)
		require.NoError(t, err)
		assert.Len(t, actual, len(testObjectID))
		assert.EqualValues(t, testObjectID, actual)
	})

	t.Run(`create settings for the given monitored entity id`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, http.StatusOK))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.CreateOrUpdateKubernetesSetting(ctx, testName, testUID, testScope)

		// assert
		require.NotNil(t, actual)
		require.NoError(t, err)
		assert.Len(t, actual, len(testObjectID))
		assert.EqualValues(t, testObjectID, actual)
	})

	t.Run(`don't create settings for the given monitored entity id because no kube-system uuid is provided`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, http.StatusOK))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.CreateOrUpdateKubernetesSetting(ctx, testName, "", testScope)

		// assert
		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run(`don't create settings for the given monitored entity id because of api error`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, http.StatusBadRequest))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.CreateOrUpdateKubernetesSetting(ctx, testName, testUID, testScope)

		// assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), strconv.Itoa(http.StatusBadRequest))
		assert.Empty(t, actual)
	})

	t.Run(`don't create settings for the given monitored entity id because of api error`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, http.StatusNotFound))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.CreateOrUpdateKubernetesSetting(ctx, testName, testUID, testScope)

		// assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), strconv.Itoa(http.StatusNotFound))
		assert.Empty(t, actual)
	})
}

func TestDynatraceClient_CreateOrUpdateAppKubernetesSetting(t *testing.T) {
	ctx := context.Background()

	t.Run(`create app settings with monitoring for the given monitored entity id`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, http.StatusOK))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.CreateOrUpdateKubernetesAppSetting(ctx, testScope)

		// assert
		require.NotNil(t, actual)
		require.NoError(t, err)
		assert.Len(t, actual, len(testObjectID))
		assert.EqualValues(t, testObjectID, actual)
	})

	t.Run(`don't create app settings for the given monitored entity id because of api error`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, http.StatusNotFound))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.CreateOrUpdateKubernetesAppSetting(ctx, testScope)

		// assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), strconv.Itoa(http.StatusNotFound))
		assert.Empty(t, actual)
	})
}

func TestDynatraceClient_getKubernetesSettingBody(t *testing.T) {
	t.Run(`get k8s settings request body for Hierarchical Monitoring Settings`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, "", http.StatusBadRequest))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual := createV3KubernetesSettingsBody(testName, testUID, testScope)

		// assert
		require.NotNil(t, actual)
		assert.Len(t, actual, 1)
		assert.EqualValues(t, hierarchicalMonitoringSettingsSchemaVersion, actual[0].SchemaVersion)
		assert.IsType(t, postKubernetesSettings{}, actual[0].Value)
		assert.True(t, actual[0].Value.(postKubernetesSettings).Enabled)
		bodyJson, err := json.Marshal(actual[0])
		require.NoError(t, err)
		assert.NotContains(t, string(bodyJson), "cloudApplicationPipelineEnabled")
		assert.NotContains(t, string(bodyJson), "openMetricsPipelineEnabled")
		assert.NotContains(t, string(bodyJson), "eventProcessingActive")
		assert.NotContains(t, string(bodyJson), "eventProcessingV2Active")
		assert.NotContains(t, string(bodyJson), "filterEvents")
		assert.Contains(t, string(bodyJson), "clusterIdEnabled")
	})

	t.Run(`get k8s settings request body`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, "", http.StatusBadRequest))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual := createV1KubernetesSettingsBody(testName, testUID, testScope)

		// assert
		require.NotNil(t, actual)
		assert.Len(t, actual, 1)
		assert.EqualValues(t, schemaVersionV1, actual[0].SchemaVersion)
		assert.IsType(t, postKubernetesSettings{}, actual[0].Value)
		assert.True(t, actual[0].Value.(postKubernetesSettings).Enabled)
		bodyJson, err := json.Marshal(actual[0])
		require.NoError(t, err)
		assert.Contains(t, string(bodyJson), "cloudApplicationPipelineEnabled")
		assert.Contains(t, string(bodyJson), "openMetricsPipelineEnabled")
		assert.Contains(t, string(bodyJson), "eventProcessingActive")
		assert.Contains(t, string(bodyJson), "eventProcessingV2Active")
		assert.Contains(t, string(bodyJson), "filterEvents")

		assert.Contains(t, string(bodyJson), "clusterIdEnabled")
	})
}

func createMonitoredEntitiesForTesting() []MonitoredEntity {
	return []MonitoredEntity{
		{EntityId: "KUBERNETES_CLUSTER-0E30FE4BF2007587", DisplayName: "operator test entity 1", LastSeenTms: 1639483869085},
		{EntityId: "KUBERNETES_CLUSTER-119C75CCDA94799F", DisplayName: "operator test entity 2", LastSeenTms: 1639034988126},
	}
}

func mockHandleEntitiesRequest(request *http.Request, writer http.ResponseWriter, entities []MonitoredEntity) {
	if request.Method == http.MethodGet {
		if !strings.Contains(request.Form.Get("entitySelector"), "type(KUBERNETES_CLUSTER)") {
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		meResponse := monitoredEntitiesResponse{
			TotalCount: len(entities),
			PageSize:   500,
			Entities:   entities,
		}

		entitiesResponse, err := json.Marshal(meResponse)
		if err != nil {
			return
		}

		writer.WriteHeader(http.StatusOK)
		writer.Write(entitiesResponse)
	} else {
		writeError(writer, http.StatusMethodNotAllowed)
	}
}

func mockHandleSettingsRequest(request *http.Request, writer http.ResponseWriter, totalCount int, objectId string) {
	switch request.Method {
	case http.MethodGet:
		if request.Form.Get("schemaIds") != "builtin:cloud.kubernetes" || request.Form.Get("scopes") == "" {
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		settingsGetResponse, err := json.Marshal(GetSettingsResponse{TotalCount: totalCount})
		if err != nil {
			return
		}

		writer.WriteHeader(http.StatusOK)
		writer.Write(settingsGetResponse)
	case http.MethodPost:
		if request.Body == nil {
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			return
		}

		var parsedBody []postKubernetesSettingsBody

		err = json.Unmarshal(body, &parsedBody)
		if err != nil {
			return
		}

		var settingsPostResponse []postSettingsResponse
		settingsPostResponse = append(settingsPostResponse, postSettingsResponse{
			ObjectId: objectId,
		})

		settingsPostResponseBytes, err := json.Marshal(settingsPostResponse)
		if err != nil {
			return
		}

		writer.WriteHeader(http.StatusOK)
		writer.Write(settingsPostResponseBytes)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}

func mockDynatraceServerEntitiesHandler(entities []MonitoredEntity, isError bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isError {
			writeError(w, http.StatusBadRequest)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		if r.FormValue("Api-Token") == "" && r.Header.Get("Authorization") == "" {
			writeError(w, http.StatusUnauthorized)
		} else {
			switch r.URL.Path {
			case "/v2/entities":
				mockHandleEntitiesRequest(r, w, entities)
			default:
				writeError(w, http.StatusBadRequest)
			}
		}
	}
}

func mockDynatraceServerSettingsHandler(totalCount int, objectId string, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			writeError(w, status)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		if r.FormValue("Api-Token") == "" && r.Header.Get("Authorization") == "" {
			writeError(w, http.StatusUnauthorized)
		} else {
			switch r.URL.Path {
			case "/v2/settings/objects":
				mockHandleSettingsRequest(r, w, totalCount, objectId)
			default:
				writeError(w, http.StatusBadRequest)
			}
		}
	}
}
