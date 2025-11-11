package dynatrace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHostEntityIDForIP(t *testing.T) {
	mockHostEntityAPI := func(status int, expectedHostInfo ...hostInfoResponse) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.FormValue("Api-Token") == "" && r.Header.Get("Authorization") == "" {
				writeError(w, http.StatusUnauthorized)
			}

			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed)
			}

			switch r.URL.Path {
			case "/v1/entity/infrastructure/hosts":
				if status != http.StatusOK {
					writeError(w, status)

					return
				}

				getResponseBytes, err := json.Marshal(expectedHostInfo)
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
	t.Run("happy path", func(t *testing.T) {
		ctx := t.Context()
		testEntities := []hostInfoResponse{
			{
				EntityID:      "HOST-42",
				NetworkZoneID: "default",
				IPAddresses:   []string{"1.1.1.1"},
			},
			{
				EntityID:      "HOST-11",
				NetworkZoneID: "default",
				IPAddresses:   []string{"1.1.1.2"},
			},
		}

		dynatraceServer := httptest.NewServer(mockHostEntityAPI(http.StatusOK, testEntities...))

		dtc := dynatraceClient{
			apiToken:   apiToken,
			paasToken:  paasToken,
			httpClient: dynatraceServer.Client(),
			url:        dynatraceServer.URL,
		}

		id, err := dtc.GetHostEntityIDForIP(ctx, "1.1.1.1")
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, "HOST-42", id)
	})

	t.Run("host entity not found path", func(t *testing.T) {
		ctx := t.Context()
		testEntities := []hostInfoResponse{
			{
				EntityID:      "HOST-11",
				NetworkZoneID: "default",
				IPAddresses:   []string{"1.1.1.2"},
			},
		}

		dynatraceServer := httptest.NewServer(mockHostEntityAPI(http.StatusOK, testEntities...))

		dtc := dynatraceClient{
			apiToken:   apiToken,
			paasToken:  paasToken,
			httpClient: dynatraceServer.Client(),
			url:        dynatraceServer.URL,
		}

		id, err := dtc.GetHostEntityIDForIP(ctx, "1.1.1.1")
		require.Error(t, err)
		assert.Empty(t, id)
	})

	t.Run("server error", func(t *testing.T) {
		ctx := t.Context()
		testEntities := []hostInfoResponse{}

		dynatraceServer := httptest.NewServer(mockHostEntityAPI(http.StatusBadGateway, testEntities...))

		dtc := dynatraceClient{
			apiToken:   apiToken,
			paasToken:  paasToken,
			httpClient: dynatraceServer.Client(),
			url:        dynatraceServer.URL,
		}

		id, err := dtc.GetHostEntityIDForIP(ctx, "1.1.1.1")
		require.Error(t, err)
		assert.Empty(t, id)
	})

	t.Run("api not found 404 error", func(t *testing.T) {
		ctx := t.Context()
		testEntities := []hostInfoResponse{}

		dynatraceServer := httptest.NewServer(mockHostEntityAPI(http.StatusNotFound, testEntities...))

		dtc := dynatraceClient{
			apiToken:   apiToken,
			paasToken:  paasToken,
			httpClient: dynatraceServer.Client(),
			url:        dynatraceServer.URL,
		}

		_, err := dtc.GetHostEntityIDForIP(ctx, "1.1.1.1")
		require.ErrorAs(t, err, &V1HostEntityAPINotAvailableErr{})
	})
}
