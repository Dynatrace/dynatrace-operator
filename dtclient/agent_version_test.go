package dtclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	apiToken  = "some-API-token"
	paasToken = "some-PaaS-token"
)

const hostsResponse = `[
  {
	"entityId": "dynatraceSampleEntityId",
    "displayName": "good",
    "lastSeenTimestamp": 1521540000000,
    "ipAddresses": [
      "10.11.12.13",
      "192.168.0.1"
    ],
    "agentVersion": {
      "major": 1,
      "minor": 142,
      "revision": 0,
      "timestamp": "20180313-173634"
    }
  },
  {
    "entityId": "unsetAgentHost",
    "displayName": "unset version",
    "ipAddresses": [
      "192.168.100.1"
    ]
  }
]`

func TestResponseForLatestVersion(t *testing.T) {
	dc := &dynatraceClient{
		logger: consoleLogger,
	}
	readFromString := func(json string) (string, error) {
		r := []byte(json)
		return dc.readResponseForLatestVersion(r)
	}

	{
		m, err := readFromString(`{"latestAgentVersion": "17"}`)
		if assert.NoError(t, err) {
			assert.Equal(t, "17", m)
		}
	}
	{
		m, err := readFromString(`{"latestAgentVersion": "179.786.861", "extraParam" : "tobeignored"}`)
		if assert.NoError(t, err) {
			assert.Equal(t, "179.786.861", m)
		}
	}
	{
		_, err := readFromString("{}")
		assert.Error(t, err, "empty response")
	}
	{
		_, err := readFromString(`{"wrong_json": ["shouldnotbeparsed"]}`)
		assert.Error(t, err, "invalid data")
	}

}

func TestGetEntityIDForIP(t *testing.T) {
	dynatraceServer, _ := createTestDynatraceClient(t, &ipHandler{})
	defer dynatraceServer.Close()

	dtc := dynatraceClient{
		logger:     log.Log.WithName("dtc"),
		apiToken:   apiToken,
		paasToken:  paasToken,
		httpClient: dynatraceServer.Client(),
		url:        dynatraceServer.URL,
	}
	require.NoError(t, dtc.setHostCacheFromResponse([]byte(
		fmt.Sprintf(`[
	{
		"entityId": "HOST-42",
		"displayName": "A",
		"firstSeenTimestamp": 1589940921731,
		"lastSeenTimestamp": %v,
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
	}
]`, time.Now().UTC().Unix()*1000))))
	id, err := dtc.GetEntityIDForIP("1.1.1.1")
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, "HOST-42", id)

	id, err = dtc.GetEntityIDForIP("2.2.2.2")

	assert.Error(t, err)
	assert.Empty(t, id)

	require.NoError(t, dtc.setHostCacheFromResponse([]byte(
		fmt.Sprintf(`[
	{
		"entityId": "",
		"displayName": "A",
		"firstSeenTimestamp": 1589940921731,
		"lastSeenTimestamp": %v,
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
	}
]`, time.Now().UTC().Unix()*1000))))

	id, err = dtc.GetEntityIDForIP("1.1.1.1")

	assert.Error(t, err)
	assert.Empty(t, id)
}

func testAgentVersionGetLatestAgentVersion(t *testing.T, dynatraceClient Client) {
	{
		_, err := dynatraceClient.GetLatestAgentVersion("", InstallerTypeDefault)

		assert.Error(t, err, "empty OS")
	}
	{
		_, err := dynatraceClient.GetLatestAgentVersion(OsUnix, "")

		assert.Error(t, err, "empty installer type")
	}
	{
		latestAgentVersion, err := dynatraceClient.GetLatestAgentVersion(OsUnix, InstallerTypeDefault)

		assert.NoError(t, err)
		assert.Equal(t, "17", latestAgentVersion, "latest agent version equals expected version")
	}
}

type ipHandler struct{}

func (ipHandler *ipHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case "GET":
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(hostsResponse))
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}

func handleLatestAgentVersion(request *http.Request, writer http.ResponseWriter) {
	switch request.Method {
	case "GET":
		writer.WriteHeader(http.StatusOK)
		out, _ := json.Marshal(map[string]string{"latestAgentVersion": "17"})
		_, _ = writer.Write(out)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}

//TODO test GetLatestAgent
