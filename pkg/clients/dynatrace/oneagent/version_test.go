package oneagent

import (
	"bytes"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

//const (
//	agentVersionHostsResponse = `[
// {
//	"entityId": "dynatraceSampleEntityId",
//   "displayName": "good",
//   "lastSeenTimestamp": 1521540000000,
//   "ipAddresses": [
//     "10.11.12.13",
//     "192.168.0.1"
//   ],
//   "agentVersion": {
//     "major": 1,
//     "minor": 142,
//     "revision": 0,
//     "timestamp": "20180313-173634"
//   }
// },
// {
//   "entityId": "unsetAgentHost",
//   "displayName": "unset version",
//   "ipAddresses": [
//     "192.168.100.1"
//   ]
// }
//]`

const (
	agentResponse          = `zip-content`
	versionedAgentResponse = `zip-content-1.2.3`
)

var versionsResponse = []string{"1.123.1", "1.123.2", "1.123.3", "1.123.4"}

func TestGetLatest(t *testing.T) {
	setupClient := func(t *testing.T, rawResponse []byte, rawErr error) (*Client, *os.File) {
		file, err := os.CreateTemp(t.TempDir(), "installer")
		require.NoError(t, err)

		req := coremock.NewAPIRequest(t)
		req.EXPECT().WithQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithRawQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithHeader("Accept", "application/octet-stream").Return(req).Once()
		req.EXPECT().ExecuteRaw().Return(rawResponse, rawErr).Once()

		client := coremock.NewAPIClient(t)
		client.EXPECT().GET(t.Context(), getLatestURL(installer.OsUnix, installer.TypePaaS)).Return(req).Once()

		return NewClient(client, ""), file
	}

	t.Run("file download successful", func(t *testing.T) {
		oaClient, file := setupClient(t, []byte(agentResponse), nil)
		err := oaClient.GetLatest(t.Context(), installer.OsUnix, installer.TypePaaS, arch.FlavorMultidistro, "arch", nil, false, file)
		require.NoError(t, err)

		resp, err := os.ReadFile(file.Name())
		require.NoError(t, err)
		assert.Equal(t, agentResponse, string(resp))
	})

	t.Run("agent not found error", func(t *testing.T) {
		oaClient, file := setupClient(t, nil, &core.HTTPError{StatusCode: 404, Message: "Not found"})
		err := oaClient.GetLatest(t.Context(), installer.OsUnix, installer.TypePaaS, arch.FlavorMultidistro, "arch", nil, false, file)
		require.Error(t, err)
	})
}

func TestGet(t *testing.T) {
	setupClient := func(t *testing.T, rawResponse []byte, rawErr error) *Client {
		req := coremock.NewAPIRequest(t)
		req.EXPECT().WithQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithRawQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithHeader(mock.Anything, mock.Anything).Return(req).Once()
		req.EXPECT().ExecuteRaw().Return(rawResponse, rawErr).Once()

		client := coremock.NewAPIClient(t)
		client.EXPECT().GET(t.Context(), getURL(installer.OsUnix, installer.TypePaaS, "")).Return(req).Once()

		return NewClient(client, "")
	}

	t.Run("handle response correctly", func(t *testing.T) {
		oaClient := setupClient(t, []byte(versionedAgentResponse), nil)
		readWriter := bytes.NewBuffer([]byte{})
		err := oaClient.Get(t.Context(), installer.OsUnix, installer.TypePaaS, "", "", "", nil, false, readWriter)

		require.NoError(t, err)
		assert.Equal(t, versionedAgentResponse, readWriter.String())
	})

	t.Run("handle server error", func(t *testing.T) {
		oaClient := setupClient(t, nil, &core.HTTPError{StatusCode: 404, Message: "Not found"})
		readWriter := bytes.NewBuffer([]byte{})
		err := oaClient.Get(t.Context(), installer.OsUnix, installer.TypePaaS, "", "", "", nil, false, readWriter)

		require.True(t, core.IsNotFound(err))
	})
}

func TestGetVersions(t *testing.T) {
	setupClient := func(t *testing.T, execErr error) *Client {
		var resp VersionsResponse

		req := coremock.NewAPIRequest(t)
		req.EXPECT().WithQueryParams(mock.Anything).Return(req).Once()
		req.EXPECT().WithPaasToken().Return(req).Once()
		req.EXPECT().Execute(&resp).Run(func(model any) {
			if execErr == nil {
				resp := model.(*VersionsResponse)
				resp.AvailableVersions = versionsResponse
			}
		}).Return(execErr).Once()

		client := coremock.NewAPIClient(t)
		client.EXPECT().GET(t.Context(), getVersionsURL(installer.OsUnix, installer.TypePaaS)).Return(req).Once()

		return NewClient(client, "")
	}

	t.Run("handle response correctly", func(t *testing.T) {
		oaClient := setupClient(t, nil)
		availableVersions, err := oaClient.GetVersions(t.Context(), installer.OsUnix, installer.TypePaaS, "")

		require.NoError(t, err)
		assert.Len(t, availableVersions, 4)
		assert.Contains(t, availableVersions, "1.123.1")
		assert.Contains(t, availableVersions, "1.123.2")
		assert.Contains(t, availableVersions, "1.123.3")
		assert.Contains(t, availableVersions, "1.123.4")
	})

	t.Run("handle server error", func(t *testing.T) {
		oaClient := setupClient(t, &core.HTTPError{StatusCode: 400, Message: "test-error"})
		availableVersions, err := oaClient.GetVersions(t.Context(), installer.OsUnix, installer.TypePaaS, "")

		require.Empty(t, availableVersions)
		require.Error(t, err)
		require.True(t, core.IsBadRequest(err))
	})
}
