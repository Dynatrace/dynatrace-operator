package dtclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	clusterVersion = "1.203.0.20200908-220956"
)

func TestDynatraceClient_GetClusterVersion_MockEndpoint(t *testing.T) {
	t.Run("getClusterVersion", func(t *testing.T) {
		dynatraceServerMock := httptest.NewServer(clusterVersionRequestHandler(handleClusterVersionRequest))
		dtc, err := NewClient(dynatraceServerMock.URL, apiToken, paasToken)

		assert.NoError(t, err)
		clusterInfo, err := dtc.GetClusterInfo()

		assert.NoError(t, err)
		assert.NotNil(t, clusterInfo)
		assert.Equal(t, clusterVersion, clusterInfo.Version)
	})

	t.Run("getClusterVersion generates error", func(t *testing.T) {
		dynatraceServerMock := httptest.NewServer(clusterVersionRequestHandler(handleRequestWithError))
		dtc, err := NewClient(dynatraceServerMock.URL, apiToken, paasToken)

		assert.NoError(t, err)
		clusterInfo, err := dtc.GetClusterInfo()

		assert.Error(t, err)
		assert.Nil(t, clusterInfo)
	})
}

func handleRequestWithError(_ *http.Request, writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusInternalServerError)
	// Suppress bytes written and error
	_, _ = writer.Write([]byte("\n"))
}

func clusterVersionRequestHandler(handlerFunc func(request *http.Request, writer http.ResponseWriter)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.FormValue("Api-Token") == "" && r.Header.Get("Authorization") == "" {
			writeError(w, http.StatusUnauthorized)
		} else {
			handlerFunc(r, w)
		}
	}
}

func handleClusterVersionRequest(request *http.Request, writer http.ResponseWriter) {
	if request.URL.Path == clusterVersionEndpoint {
		data := ClusterInfo{Version: clusterVersion}
		rawData, err := json.Marshal(&data)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			// Suppress bytes written and error
			_, _ = writer.Write([]byte("\n"))
		}
		// Suppress bytes written and error
		_, _ = writer.Write(rawData)
	} else {
		writer.WriteHeader(http.StatusInternalServerError)
		// Suppress bytes written and error
		_, _ = writer.Write([]byte("\n"))
	}
}
