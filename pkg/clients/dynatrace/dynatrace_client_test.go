package dynatrace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeRequest(t *testing.T) {
	ctx := context.Background()

	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	dc := &dynatraceClient{
		url:       dynatraceServer.URL,
		apiToken:  apiToken,
		paasToken: paasToken,

		hostCache:  make(map[string]hostInfo),
		httpClient: http.DefaultClient,
	}

	require.NotNil(t, dc)
	t.Run("happy path", func(t *testing.T) {
		url := fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dc.url)
		resp, err := dc.makeRequest(ctx, url, dynatraceApiToken)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		defer utils.CloseBodyAfterRequest(resp)
	})

	t.Run("sad path", func(t *testing.T) {
		resp, err := dc.makeRequest(ctx, "%s/v1/deployment/installer/agent/connectioninfo", dynatraceApiToken)
		require.Error(t, err, "unsupported protocol scheme")
		assert.Nil(t, resp)
	})
}

func TestGetResponseOrServerError(t *testing.T) {
	ctx := context.Background()

	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	dc := &dynatraceClient{
		url:       dynatraceServer.URL,
		apiToken:  apiToken,
		paasToken: paasToken,

		hostCache:  make(map[string]hostInfo),
		httpClient: http.DefaultClient,
	}

	require.NotNil(t, dc)
	t.Run("happy path", func(t *testing.T) {
		reqURL := fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dc.url)
		resp, err := dc.makeRequest(ctx, reqURL, dynatraceApiToken)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		body, err := dc.getServerResponseData(resp)
		require.NoError(t, err)
		assert.NotNil(t, body, "response body available")
	})
}

func TestBuildHostCache(t *testing.T) {
	ctx := context.Background()

	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	dc := &dynatraceClient{
		url:       dynatraceServer.URL,
		paasToken: paasToken,
		now:       time.Unix(1521540000, 0),

		hostCache:  make(map[string]hostInfo),
		httpClient: http.DefaultClient,
	}

	require.NotNil(t, dc)
	t.Run("sad path", func(t *testing.T) {
		dc.apiToken = ""
		err := dc.buildHostCache(ctx)
		require.Error(t, err, "error querying dynatrace server")
		assert.Empty(t, dc.hostCache)
	})
	t.Run("happy path", func(t *testing.T) {
		dc.apiToken = apiToken
		err := dc.buildHostCache(ctx)
		require.NoError(t, err)
		assert.NotZero(t, len(dc.hostCache))
		assert.ObjectsAreEqualValues(dc.hostCache, map[string]hostInfo{
			"10.11.12.13": {version: "1.142.0.20180313-173634", entityID: "dynatraceSampleEntityId"},
			"192.168.0.1": {version: "1.142.0.20180313-173634", entityID: "dynatraceSampleEntityId"},
		})
	})
}

func TestServerError(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		se := &ServerError{Code: 401, Message: "Unauthorized"}
		assert.Equal(t, "dynatrace server error 401: Unauthorized", se.Error())
	})
	t.Run("no status code is required", func(t *testing.T) {
		se := &ServerError{Message: "Unauthorized"}
		assert.Equal(t, "dynatrace server error 0: Unauthorized", se.Error())
	})
	t.Run("no message is required", func(t *testing.T) {
		se := &ServerError{Code: 401}
		assert.Equal(t, "dynatrace server error 401: ", se.Error())
	})
	t.Run("unknown error", func(t *testing.T) {
		se := &ServerError{}
		assert.Equal(t, "unknown server error", se.Error())
	})
}

func TestDynatraceClientWithServer(t *testing.T) {
	dynatraceServer := httptest.NewServer(dynatraceServerHandler())
	defer dynatraceServer.Close()

	skipCert := SkipCertificateValidation(true)
	dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)
	dtc.(*dynatraceClient).now = time.Unix(1521540000, 0)

	require.NoError(t, err)
	require.NotNil(t, dtc)

	// TODO: Fix this monster, this is not ok
	testAgentVersionGetLatestAgentVersion(t, dtc)
	testActiveGateVersionGetLatestActiveGateVersion(t, dtc)
	testCommunicationHostsGetCommunicationHosts(t, dtc)
	testSendEvent(t, dtc)
	testGetTokenScopes(t, dtc)
}

func dynatraceServerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.FormValue("Api-Token") == "" && r.Header.Get("Authorization") == "" {
			writeError(w, http.StatusUnauthorized)
		} else {
			handleRequest(r, w)
		}
	}
}

func handleRequest(request *http.Request, writer http.ResponseWriter) {
	latestAgentVersion := fmt.Sprintf("/v1/deployment/installer/agent/%s/%s/latest/metainfo", OsUnix, InstallerTypePaaS)
	latestActiveGateVersion := fmt.Sprintf("/v1/deployment/installer/gateway/%s/latest/metainfo", OsUnix)
	agentVersions := fmt.Sprintf("/v1/deployment/installer/agent/versions/%s/%s", OsUnix, InstallerTypePaaS)

	switch request.URL.Path {
	case latestAgentVersion:
		handleLatestAgentVersion(request, writer)
	case latestActiveGateVersion:
		handleLatestActiveGateVersion(request, writer)
	case agentVersions:
		handleAvailableAgentVersions(request, writer)
	case "/v1/entity/infrastructure/hosts":
		(&ipHandler{}).ServeHTTP(writer, request)
	case "/v1/deployment/installer/agent/connectioninfo":
		handleCommunicationHosts(request, writer)
	case "/v1/events":
		handleSendEvent(request, writer)
	case "/v1/tokens/lookup":
		handleTokenScopes(request, writer)
	default:
		writeError(writer, http.StatusBadRequest)
	}
}

func writeError(w http.ResponseWriter, status int) {
	message := serverErrorResponse{
		ErrorMessage: ServerError{
			Code:    status,
			Message: "error received from server",
		},
	}
	result, _ := json.Marshal(&message)

	w.WriteHeader(status)
	_, _ = w.Write(result)
}

func TestIgnoreNonCurrentlySeenHosts(t *testing.T) {
	ctx := context.Background()
	// now:                         20/05/2020 10:10 AM UTC
	// HOST-42 - lastSeenTimestamp: 20/05/2020 10:04 AM UTC
	// HOST-84 - lastSeenTimestamp: 19/05/2020 01:49 AM UTC
	c := dynatraceClient{
		now: time.Unix(1589969400, 0).UTC(),
	}

	require.NoError(t, c.setHostCacheFromResponse([]byte(`[
	{
		"entityId": "HOST-42",
		"displayName": "A",
		"firstSeenTimestamp": 1589940921731,
		"lastSeenTimestamp": 1589969061511,
		"ipAddresses": [
			"1.1.1.1"
		],
		"monitoringMode": "FULL_STACK",
		"networkZoneId": "default",
		"agentVersion": {
			"major": 1,
			"minor": 195,
			"revision": 0,
			"timestamp": "20200515-045253",
			"sourceRevision": ""
		}
	},
	{
		"entityId": "HOST-84",
		"displayName": "B",
		"firstSeenTimestamp": 1589767448722,
		"lastSeenTimestamp": 1589852948530,
		"ipAddresses": [
			"1.1.1.1"
		],
		"monitoringMode": "FULL_STACK",
		"networkZoneId": "default"
	}
]`)))

	info, err := c.getHostInfoForIP(ctx, "1.1.1.1")
	require.NoError(t, err)
	require.Equal(t, "HOST-42", info.entityID)
	require.Equal(t, "1.195.0.20200515-045253", info.version)
}

func createTestDynatraceServer(t *testing.T, handler http.Handler, networkZoneName string) (*httptest.Server, Client) {
	dynatraceServer := httptest.NewServer(handler)

	skipCert := SkipCertificateValidation(true)
	networkZone := NetworkZone(networkZoneName)
	dynatraceClient, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert, networkZone)

	require.NoError(t, err)
	require.NotNil(t, dynatraceClient)

	return dynatraceServer, dynatraceClient
}

func createTestDynatraceClientWithFunc(t *testing.T, handler http.HandlerFunc) (*httptest.Server, Client) {
	dynatraceServer := httptest.NewServer(handler)

	skipCert := SkipCertificateValidation(true)
	dynatraceClient, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)

	require.NoError(t, err)
	require.NotNil(t, dynatraceClient)

	return dynatraceServer, dynatraceClient
}
