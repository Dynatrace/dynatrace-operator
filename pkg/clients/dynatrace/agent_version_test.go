package dynatrace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
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
	ctx := context.Background()

	dynatraceServer, _ := createTestDynatraceServer(t, &ipHandler{}, "")
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

	id, err := dtc.GetEntityIDForIP(ctx, "1.1.1.1")
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, "HOST-42", id)

	id, err = dtc.GetEntityIDForIP(ctx, "2.2.2.2")

	require.Error(t, err)
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

	id, err = dtc.GetEntityIDForIP(ctx, "1.1.1.1")

	require.Error(t, err)
	assert.Empty(t, id)
}

func testAgentVersionGetLatestAgentVersion(t *testing.T, dynatraceClient Client) {
	ctx := context.Background()

	t.Run("os field is required", func(t *testing.T) {
		_, err := dynatraceClient.GetLatestAgentVersion(ctx, "", InstallerTypeDefault)

		require.Error(t, err, "empty OS")
	})

	t.Run("installer field is required", func(t *testing.T) {
		_, err := dynatraceClient.GetLatestAgentVersion(ctx, OsUnix, "")

		require.Error(t, err, "empty installer type")
	})

	t.Run("happy path", func(t *testing.T) {
		latestAgentVersion, err := dynatraceClient.GetLatestAgentVersion(ctx, OsUnix, InstallerTypePaaS)

		require.NoError(t, err)
		assert.Equal(t, "1.242.0.20220429-180918", latestAgentVersion, "latest agent version equals expected version")
	})
}

func TestGetLatestAgent(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()

	dynatraceServer, _ := createTestDynatraceServer(t, &ipHandler{fs}, "")
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

		err = dtc.GetLatestAgent(ctx, OsUnix, InstallerTypePaaS, arch.FlavorMultidistro, "arch", nil, false, file)
		require.NoError(t, err)

		resp, err := afero.ReadFile(fs, file.Name())
		require.NoError(t, err)

		assert.Equal(t, agentResponse, string(resp))
	})
	t.Run(`missing agent error`, func(t *testing.T) {
		file, err := afero.TempFile(fs, "client", "installer")
		require.NoError(t, err)

		err = dtc.GetLatestAgent(ctx, OsUnix, InstallerTypePaaS, arch.FlavorMultidistro, "invalid", nil, false, file)
		require.Error(t, err)
	})
}

func TestDynatraceClient_GetAgent(t *testing.T) {
	ctx := context.Background()

	t.Run(`handle response correctly`, func(t *testing.T) {
		dynatraceServer, dtc := createTestDynatraceClientWithFunc(t, agentRequestHandler)
		defer dynatraceServer.Close()

		readWriter := &memoryReadWriter{data: make([]byte, len(versionedAgentResponse))}
		err := dtc.GetAgent(ctx, OsUnix, InstallerTypePaaS, "", "", "", nil, false, readWriter)

		require.NoError(t, err)
		assert.Equal(t, versionedAgentResponse, string(readWriter.data))
	})
	t.Run(`handle server error`, func(t *testing.T) {
		dynatraceServer, dtc := createTestDynatraceClientWithFunc(t, errorHandler)
		defer dynatraceServer.Close()

		readWriter := &memoryReadWriter{data: make([]byte, len(versionedAgentResponse))}
		err := dtc.GetAgent(ctx, OsUnix, InstallerTypePaaS, "", "", "", nil, false, readWriter)

		require.EqualError(t, err, "dynatrace server error 400: test-error")
	})
}

func TestDynatraceClient_GetAgentVersions(t *testing.T) {
	ctx := context.Background()

	t.Run(`handle response correctly`, func(t *testing.T) {
		dynatraceServer, dtc := createTestDynatraceClientWithFunc(t, versionsRequestHandler)
		defer dynatraceServer.Close()

		availableVersions, err := dtc.GetAgentVersions(ctx, OsUnix, InstallerTypePaaS, "")

		require.NoError(t, err)
		assert.Len(t, availableVersions, 4)
		assert.Contains(t, availableVersions, "1.123.1")
		assert.Contains(t, availableVersions, "1.123.2")
		assert.Contains(t, availableVersions, "1.123.3")
		assert.Contains(t, availableVersions, "1.123.4")
	})
	t.Run(`handle server error`, func(t *testing.T) {
		dynatraceServer, dtc := createTestDynatraceClientWithFunc(t, errorHandler)
		defer dynatraceServer.Close()

		availableVersions, err := dtc.GetAgentVersions(ctx, OsUnix, InstallerTypePaaS, "")

		require.EqualError(t, err, "dynatrace server error 400: test-error")
		assert.Empty(t, availableVersions)
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
	case http.MethodGet:
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
	case http.MethodGet:
		writer.WriteHeader(http.StatusOK)

		out, _ := json.Marshal(map[string]string{"latestAgentVersion": "1.242.0.20220429-180918"})
		_, _ = writer.Write(out)
	default:
		writeError(writer, http.StatusMethodNotAllowed)
	}
}

func handleAvailableAgentVersions(request *http.Request, writer http.ResponseWriter) {
	switch request.Method {
	case http.MethodGet:
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
