package dynatrace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDynatraceClient_CreateOrUpdateKubernetesSetting(t *testing.T) {
	ctx := context.Background()

	t.Run(`create settings with monitoring for the given monitored entity id`, func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, testObjectID, http.StatusOK)))
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
		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, testObjectID, http.StatusOK)))
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
		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, testObjectID, http.StatusOK)))
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
		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, testObjectID, http.StatusBadRequest)))
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
		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, testObjectID, http.StatusNotFound)))
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
		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, testObjectID, http.StatusOK)))
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
		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, testObjectID, http.StatusNotFound)))
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
		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, "", http.StatusBadRequest)))
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
		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(1, "", http.StatusBadRequest)))
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

type v2APIMockParams struct {
	entitiesAPI entitiesMockParams
	settingsAPI settingsMockParams
}

type entitiesMockParams struct {
	status   int
	expected []MonitoredEntity
}

type settingsMockParams struct {
	status     int
	totalCount int
	objectID   string
}

func createKubernetesSettingsMockParams(totalCount int, objectID string, status int) v2APIMockParams {
	return v2APIMockParams{
		settingsAPI: settingsMockParams{
			totalCount: totalCount,
			objectID:   objectID,
			status:     status,
		},
	}
}

func createEntitiesMockParams(expected []MonitoredEntity, status int) v2APIMockParams {
	return v2APIMockParams{
		entitiesAPI: entitiesMockParams{
			status:   status,
			expected: expected,
		},
	}
}

func mockDynatraceServerV2Handler(params v2APIMockParams) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.FormValue("Api-Token") == "" && r.Header.Get("Authorization") == "" {
			writeError(w, http.StatusUnauthorized)
		} else {
			switch r.URL.Path {
			case "/v2/settings/objects":
				if params.settingsAPI.status != http.StatusOK {
					writeError(w, params.settingsAPI.status)

					return
				}

				mockHandleSettingsRequest(r, w, params.settingsAPI.totalCount, params.settingsAPI.objectID)
			case "/v2/settings/effectiveValues":
				if r.URL.Query().Get(scopeQueryParam) == "" {
					writeError(w, http.StatusBadRequest)

					return
				} else if params.settingsAPI.status != http.StatusOK {
					writeError(w, params.settingsAPI.status)

					return
				}

				mockHandleEffectiveSettingsRequest(r, w, params.settingsAPI.totalCount)
			case "/v2/entities":
				if params.entitiesAPI.status != http.StatusOK {
					writeError(w, params.entitiesAPI.status)

					return
				}

				mockHandleEntitiesRequest(r, w, params.entitiesAPI.expected)
			default:
				writeError(w, http.StatusBadRequest)
			}
		}
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
		if request.Form.Get("scopes") == "" {
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		if request.Form.Get("schemaIds") == KubernetesSettingsSchemaId {
			mockGetKubernetesSettingsAPI(writer, totalCount)
		}

	case http.MethodPost:
		mockPostKubernetesSettingAPI(request, writer, objectId)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}

func mockHandleEffectiveSettingsRequest(request *http.Request, writer http.ResponseWriter, totalCount int) {
	switch request.Method {
	case http.MethodGet:
		if request.Form.Get("scope") == "" {
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		if request.Form.Get("schemaIds") == MetadataEnrichmentSettingsSchemaId {
			mockGetRulesSettingsAPI(writer, totalCount)
		}
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}

func mockGetKubernetesSettingsAPI(writer http.ResponseWriter, totalCount int) {
	settingsGetResponse, err := json.Marshal(GetSettingsResponse{TotalCount: totalCount})
	if err != nil {
		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Write(settingsGetResponse)
}

func mockPostKubernetesSettingAPI(request *http.Request, writer http.ResponseWriter, objectId string) {
	if request.Body == nil {
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	body, err := io.ReadAll(request.Body)
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
}
