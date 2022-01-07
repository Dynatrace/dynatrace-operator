package dtclient

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
		actual, err := dtc.(*dynatraceClient).GetMonitoredEntitiesForKubeSystemUUID(testUID)

		// assert
		assert.NotNil(t, actual)
		assert.NoError(t, err)
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
		actual, err := dtc.(*dynatraceClient).GetMonitoredEntitiesForKubeSystemUUID(testUID)

		// assert
		assert.NotNil(t, actual)
		assert.NoError(t, err)
		assert.Len(t, actual, 0)
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
		actual, err := dtc.(*dynatraceClient).GetMonitoredEntitiesForKubeSystemUUID("")

		// assert
		assert.Nil(t, actual)
		assert.Error(t, err)
		assert.Len(t, actual, 0)
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
		actual, err := dtc.(*dynatraceClient).GetMonitoredEntitiesForKubeSystemUUID(testUID)

		// assert
		assert.Nil(t, actual)
		assert.Error(t, err)
		assert.Len(t, actual, 0)
	})
}

func TestDynatraceClient_GetSettingsForMonitoredEntities(t *testing.T) {
	t.Run(`settings for the given monitored entities exist`, func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()
		totalCount := 2

		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(totalCount, "", false))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.(*dynatraceClient).GetSettingsForMonitoredEntities(expected)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, actual)
		assert.True(t, actual.TotalCount > 0)
		assert.Equal(t, len(expected), actual.TotalCount)
	})

	t.Run(`no settings for the given monitored entities exist`, func(t *testing.T) {
		// arrange
		expected := createMonitoredEntitiesForTesting()
		totalCount := 0

		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(totalCount, "", false))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.(*dynatraceClient).GetSettingsForMonitoredEntities(expected)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, actual)
		assert.True(t, actual.TotalCount < 1)
	})

	t.Run(`no settings for an empty list of monitored entities exist`, func(t *testing.T) {
		// arrange
		entities := []MonitoredEntity{}
		// it is immaterial what we put here since no http call is executed when the list of
		// monitored entities is empty, therefore also no settings will be returned
		totalCount := 999

		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(totalCount, "", false))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.(*dynatraceClient).GetSettingsForMonitoredEntities(entities)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, actual)
		assert.True(t, actual.TotalCount == 0)
	})

	t.Run(`no settings found for because of an api error`, func(t *testing.T) {
		// arrange
		entities := createMonitoredEntitiesForTesting()
		// it is immaterial what we put here since the http request is producing an error
		totalCount := 999

		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(totalCount, "", true))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.(*dynatraceClient).GetSettingsForMonitoredEntities(entities)

		// assert
		assert.Error(t, err)
		assert.True(t, actual.TotalCount == 0)
	})
}

func TestDynatraceClient_CreateOrUpdateKubernetesSetting(t *testing.T) {
	t.Run(`create settings for the given monitored entity id`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, false))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.(*dynatraceClient).CreateOrUpdateKubernetesSetting(testName, testUID, testScope)

		// assert
		assert.NotNil(t, actual)
		assert.NoError(t, err)
		assert.Len(t, actual, len(testObjectID))
		assert.EqualValues(t, testObjectID, actual)
	})

	t.Run(`don't create settings for the given monitored entity id because no kube-system uuid is provided`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, false))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.(*dynatraceClient).CreateOrUpdateKubernetesSetting(testName, "", testScope)

		// assert
		assert.Error(t, err)
		assert.Len(t, actual, 0)
	})

	t.Run(`don't create settings for the given monitored entity id because of api error`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerSettingsHandler(1, testObjectID, true))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.(*dynatraceClient).CreateOrUpdateKubernetesSetting(testName, testUID, testScope)

		// assert
		assert.Error(t, err)
		assert.Len(t, actual, 0)
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
	if request.Method == http.MethodGet {
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
	} else if request.Method == http.MethodPost {
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
	} else {
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

func mockDynatraceServerSettingsHandler(totalCount int, objectId string, isError bool) http.HandlerFunc {
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
			case "/v2/settings/objects":
				mockHandleSettingsRequest(r, w, totalCount, objectId)
			default:
				writeError(w, http.StatusBadRequest)
			}
		}
	}
}
