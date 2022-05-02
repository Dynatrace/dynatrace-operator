package dtclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	apiToken  = "some-API-token"
	paasToken = "some-PaaS-token"

	testErrorMessage = `{ "error": { "message" : "test-error", "code": 400 } }`
)

const (
	agentVersionHostsResponse = `[
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

	agentResponse          = `zip-content`
	versionedAgentResponse = `zip-content-1.2.3`
	versionsResponse       = `{ "availableVersions": [ "1.123.1", "1.123.2", "1.123.3", "1.123.4" ] }`
)

func TestGetEntityIDForIP(t *testing.T) {
	dynatraceServer, _ := createTestDynatraceClient(t, &ipHandler{}, "")
	defer dynatraceServer.Close()

	dtc := dynatraceClient{
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
		latestAgentVersion, err := dynatraceClient.GetLatestAgentVersion(OsUnix, InstallerTypePaaS)

		assert.NoError(t, err)
		assert.Equal(t, "1.242.0.20220429-180918", latestAgentVersion, "latest agent version equals expected version")
	}
}

func TestGetLatestAgent(t *testing.T) {
	fs := afero.NewMemMapFs()

	dynatraceServer, _ := createTestDynatraceClient(t, &ipHandler{fs}, "")
	defer dynatraceServer.Close()

	dtc := dynatraceClient{
		apiToken:   apiToken,
		paasToken:  paasToken,
		httpClient: dynatraceServer.Client(),
		url:        dynatraceServer.URL,
	}

	t.Run(`file download successful`, func(t *testing.T) {
		file, err := afero.TempFile(fs, "client", "installer")
		require.NoError(t, err)

		err = dtc.GetLatestAgent(OsUnix, InstallerTypePaaS, arch.FlavorMultidistro, "arch", nil, file)
		require.NoError(t, err)

		resp, err := afero.ReadFile(fs, file.Name())
		require.NoError(t, err)

		assert.Equal(t, agentResponse, string(resp))
	})
	t.Run(`missing agent error`, func(t *testing.T) {
		file, err := afero.TempFile(fs, "client", "installer")
		require.NoError(t, err)

		err = dtc.GetLatestAgent(OsUnix, InstallerTypePaaS, arch.FlavorMultidistro, "invalid", nil, file)
		require.Error(t, err)
	})
}

func TestDynatraceClient_GetAgent(t *testing.T) {
	t.Run(`handle response correctly`, func(t *testing.T) {
		dynatraceServer, _ := createTestDynatraceClientWithFunc(t, agentRequestHandler)
		defer dynatraceServer.Close()

		dtc := dynatraceClient{
			httpClient: dynatraceServer.Client(),
			url:        dynatraceServer.URL,
			paasToken:  paasToken,
		}
		readWriter := &memoryReadWriter{data: make([]byte, len(versionedAgentResponse))}
		err := dtc.GetAgent(OsUnix, InstallerTypePaaS, "", "", "", nil, readWriter)

		assert.NoError(t, err)
		assert.Equal(t, versionedAgentResponse, string(readWriter.data))
	})
	t.Run(`handle server error`, func(t *testing.T) {
		dynatraceServer, _ := createTestDynatraceClientWithFunc(t, errorHandler)
		defer dynatraceServer.Close()

		dtc := dynatraceClient{
			httpClient: dynatraceServer.Client(),
			url:        dynatraceServer.URL,
			paasToken:  paasToken,
		}
		readWriter := &memoryReadWriter{data: make([]byte, len(versionedAgentResponse))}
		err := dtc.GetAgent(OsUnix, InstallerTypePaaS, "", "", "", nil, readWriter)

		assert.EqualError(t, err, "dynatrace server error 400: test-error")
	})
}

func TestDynatraceClient_GetAgentVersions(t *testing.T) {
	t.Run(`handle response correctly`, func(t *testing.T) {
		dynatraceServer, _ := createTestDynatraceClientWithFunc(t, versionsRequestHandler)
		defer dynatraceServer.Close()

		dtc := dynatraceClient{
			httpClient: dynatraceServer.Client(),
			url:        dynatraceServer.URL,
			paasToken:  paasToken,
		}
		availableVersions, err := dtc.GetAgentVersions(OsUnix, InstallerTypePaaS, "", "")

		assert.NoError(t, err)
		assert.Equal(t, 4, len(availableVersions))
		assert.Contains(t, availableVersions, "1.123.1")
		assert.Contains(t, availableVersions, "1.123.2")
		assert.Contains(t, availableVersions, "1.123.3")
		assert.Contains(t, availableVersions, "1.123.4")
	})
	t.Run(`handle server error`, func(t *testing.T) {
		dynatraceServer, _ := createTestDynatraceClientWithFunc(t, errorHandler)
		defer dynatraceServer.Close()

		dtc := dynatraceClient{
			httpClient: dynatraceServer.Client(),
			url:        dynatraceServer.URL,
			paasToken:  paasToken,
		}
		availableVersions, err := dtc.GetAgentVersions(OsUnix, InstallerTypePaaS, "", "")

		assert.EqualError(t, err, "dynatrace server error 400: test-error")
		assert.Equal(t, 0, len(availableVersions))
	})
}

func versionsRequestHandler(response http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte(versionsResponse))
	} else {
		response.WriteHeader(http.StatusBadRequest)
		_, _ = response.Write([]byte{})
	}
}

func agentRequestHandler(response http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte(versionedAgentResponse))
	} else {
		response.WriteHeader(http.StatusBadRequest)
		_, _ = response.Write([]byte{})
	}
}

func errorHandler(response http.ResponseWriter, _ *http.Request) {
	response.WriteHeader(http.StatusBadRequest)
	_, _ = response.Write([]byte(testErrorMessage))
}

type memoryReadWriter struct {
	data []byte
}

func (m *memoryReadWriter) Read(p []byte) (n int, err error) {
	return copy(p, m.data), nil
}

func (m *memoryReadWriter) Write(p []byte) (n int, err error) {
	return copy(m.data, p), nil
}

type ipHandler struct {
	fs afero.Fs
}

func (ipHandler *ipHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	arch, present := query["arch"]
	if present && arch[0] == "invalid" {
		writeError(writer, http.StatusNotFound)
		return
	}

	switch request.Method {
	case "GET":
		writer.WriteHeader(http.StatusOK)
		resp := []byte(agentVersionHostsResponse)
		if strings.HasSuffix(request.URL.Path, "/latest") {
			// write to temp file and write content to response
			writer.Header().Set("Content-Type", "application/octet-stream")
			file, _ := afero.TempFile(ipHandler.fs, "server", "installer")
			_, _ = file.WriteString(agentResponse)

			resp, _ = afero.ReadFile(ipHandler.fs, file.Name())
		}
		_, _ = writer.Write(resp)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}

func handleLatestAgentVersion(request *http.Request, writer http.ResponseWriter) {
	switch request.Method {
	case "GET":
		writer.WriteHeader(http.StatusOK)
		out, _ := json.Marshal(
			map[string][]string{
				"availableVersions": {
					"1.241.6.20220422-072953",
					"1.241.0.20220421-185631",
					"1.241.15.20220425-161457",
					"1.242.0.20220429-180918",
					"1.239.0.20220324-225902",
					"1.240.0.20220407-234527",
					"1.242.0.20220429-165750",
				},
			},
		)
		_, _ = writer.Write(out)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}
