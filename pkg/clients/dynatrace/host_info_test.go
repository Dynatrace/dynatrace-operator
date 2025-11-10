package dynatrace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
				EntityID:          "HOST-42",
				LastSeenTimestamp: time.Now().UTC().UnixMilli(),
				NetworkZoneID:     "default",
				IPAddresses:       []string{"1.1.1.1"},
			},
			{
				EntityID:          "HOST-11",
				LastSeenTimestamp: time.Now().UTC().UnixMilli(),
				NetworkZoneID:     "default",
				IPAddresses:       []string{"1.1.1.2"},
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

	t.Run("not found path", func(t *testing.T) {
		ctx := t.Context()
		testEntities := []hostInfoResponse{
			{
				EntityID:          "HOST-11",
				LastSeenTimestamp: time.Now().UTC().UnixMilli(),
				NetworkZoneID:     "default",
				IPAddresses:       []string{"1.1.1.2"},
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

	t.Run("ignore non currently seen hosts", func(t *testing.T) {
		ctx := t.Context()

		// now:                         20/05/2020 10:10 AM UTC
		// HOST-42 - lastSeenTimestamp: 20/05/2020 10:04 AM UTC
		// HOST-11 - lastSeenTimestamp: 19/05/2020 01:49 AM UTC
		testEntities := []hostInfoResponse{
			{
				EntityID:          "HOST-42",
				LastSeenTimestamp: 1589969061511,
				NetworkZoneID:     "default",
				IPAddresses:       []string{"1.1.1.1"},
			},
			{
				EntityID:          "HOST-11",
				LastSeenTimestamp: 1589852948530,
				NetworkZoneID:     "default",
				IPAddresses:       []string{"1.1.1.1"},
			},
		}

		dynatraceServer := httptest.NewServer(mockHostEntityAPI(http.StatusOK, testEntities...))

		dtc := dynatraceClient{
			apiToken:   apiToken,
			paasToken:  paasToken,
			httpClient: dynatraceServer.Client(),
			url:        dynatraceServer.URL,
			now:        time.Unix(1589969400, 0).UTC(),
		}

		id, err := dtc.GetHostEntityIDForIP(ctx, "1.1.1.1")
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, "HOST-42", id)
	})
}
