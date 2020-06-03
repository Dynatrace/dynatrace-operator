package dtclient

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	apiToken  = "some-API-token"
	paasToken = "some-PaaS-token"

	goodIP    = "192.168.0.1"
	unsetIP   = "192.168.100.1"
	unknownIP = "127.0.0.1"
)

const hostsResponse = `[
  {
	"entityId": "dynatraceSampleEntityId",
    "displayName": "good",
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
    "displayName": "unset version",
    "ipAddresses": [
      "192.168.100.1"
    ]
  }
]`

func TestResponseForLatestVersion(t *testing.T) {
	dc := &dynatraceClient{}
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
		_, err := readFromString("")
		assert.Error(t, err, "empty response")
	}
	{
		_, err := readFromString(`{"wrong_json": ["shouldnotbeparsed"]}`)
		assert.Error(t, err, "invalid data")
	}

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

func testAgentVersionGetAgentVersionForIP(t *testing.T, dynatraceClient Client) {
	{
		_, err := dynatraceClient.GetAgentVersionForIP("")

		assert.Error(t, err, "lookup empty ip")
	}
	{
		_, err := dynatraceClient.GetAgentVersionForIP(unknownIP)

		assert.Error(t, err, "lookup unknown ip")
	}
	{
		_, err := dynatraceClient.GetAgentVersionForIP(unsetIP)

		assert.Error(t, err, "lookup unset ip")
	}
	{
		version, err := dynatraceClient.GetAgentVersionForIP(goodIP)

		assert.NoError(t, err, "lookup good ip")
		assert.Equal(t, "1.142.0.20180313-173634", version, "version matches for lookup good ip")
	}
}

func handleVersionForIP(request *http.Request, writer http.ResponseWriter) {
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
