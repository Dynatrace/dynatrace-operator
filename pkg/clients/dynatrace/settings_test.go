package dynatrace

import (
	"context"
	"encoding/json"
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

func TestDynatraceClient_GetK8sClusterME(t *testing.T) {
	mockK8sSettingsAPI := func(status int, expectedEntity K8sClusterME) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.FormValue("Api-Token") == "" && r.Header.Get("Authorization") == "" {
				writeError(w, http.StatusUnauthorized)
			}

			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed)
			}

			switch r.URL.Path {
			case "/v2/settings/objects":
				if status != http.StatusOK {
					writeError(w, status)

					return
				}

				getResponse := getSettingsForKubeSystemUUIDResponse{
					TotalCount: 0,
					PageSize:   50,
				}

				if expectedEntity.ID != "" {
					getResponse.Settings = []kubernetesSetting{
						{
							EntityID: expectedEntity.ID,
							Value: kubernetesSettingValue{
								Label: expectedEntity.Name,
							},
						},
					}
					getResponse.TotalCount = 1
				}

				getResponseBytes, err := json.Marshal(getResponse)
				if err != nil {
					return
				}

				w.WriteHeader(http.StatusOK)
				w.Write(getResponseBytes)

			default:
				writeError(w, http.StatusBadRequest)
			}
		}
	}

	ctx := context.Background()

	t.Run("k8s entity for this uuid exist", func(t *testing.T) {
		// arrange
		expected := createKubernetesClusterEntityForTesting()

		dynatraceServer := httptest.NewServer(mockK8sSettingsAPI(http.StatusOK, expected))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetK8sClusterME(ctx, testUID)

		// assert
		require.NotNil(t, actual)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("no k8s entity for this uuid exist", func(t *testing.T) {
		// arrange
		expected := K8sClusterME{}

		dynatraceServer := httptest.NewServer(mockK8sSettingsAPI(http.StatusOK, expected))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetK8sClusterME(ctx, testUID)

		// assert
		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("no k8s entity + no API call if no kube-system uuid is provided", func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(http.NotFoundHandler())
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetK8sClusterME(ctx, "")

		// assert
		require.Error(t, err)
		assert.Empty(t, actual)
	})

	t.Run("no monitored entities found because of an api error", func(t *testing.T) {
		// arrange
		dynatraceServer := httptest.NewServer(mockK8sSettingsAPI(http.StatusInternalServerError, K8sClusterME{}))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetK8sClusterME(ctx, testUID)

		// assert
		require.Error(t, err)
		assert.Empty(t, actual)
	})
}

func TestDynatraceClient_GetSettingsForMonitoredEntity(t *testing.T) {
	ctx := context.Background()

	t.Run("settings for the given monitored entities exist", func(t *testing.T) {
		// arrange
		expected := createKubernetesClusterEntityForTesting()
		totalCount := 1

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(totalCount, "", http.StatusOK)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntity(ctx, expected, KubernetesSettingsSchemaID)

		// assert
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Positive(t, actual.TotalCount)
	})

	t.Run("no settings for the given monitored entities exist", func(t *testing.T) {
		// arrange
		expected := createKubernetesClusterEntityForTesting()
		totalCount := 0

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(totalCount, "", http.StatusOK)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		// act
		actual, err := dtc.GetSettingsForMonitoredEntity(ctx, expected, KubernetesSettingsSchemaID)

		// assert
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Less(t, actual.TotalCount, 1)
	})

	t.Run("no settings for an empty list of monitored entities exist", func(t *testing.T) {
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
		actual, err := dtc.GetSettingsForMonitoredEntity(ctx, K8sClusterME{}, KubernetesSettingsSchemaID)

		// assert
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Empty(t, actual.TotalCount)
	})

	t.Run("no settings found for because of an api error", func(t *testing.T) {
		k8sEntity := createKubernetesClusterEntityForTesting()

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(createKubernetesSettingsMockParams(-1, "", http.StatusNotFound)))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
		require.NoError(t, err)
		require.NotNil(t, dtc)

		actual, err := dtc.GetSettingsForMonitoredEntity(ctx, k8sEntity, KubernetesSettingsSchemaID)
		require.Error(t, err)
		assert.True(t, IsNotFound(err))
		assert.Empty(t, actual.TotalCount)
	})
}

func createKubernetesClusterEntityForTesting() K8sClusterME {
	return K8sClusterME{
		ID: "KUBERNETES_CLUSTER-0E30FE4BF2007587", Name: "operator test entity 1",
	}
}
