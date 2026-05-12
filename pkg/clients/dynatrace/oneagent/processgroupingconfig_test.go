package oneagent

import (
	"io"
	"net/http"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testCBORData     = "\x81\x01" // minimal CBOR: array of one integer
	testETag         = `"abc123"`
	testResponseETag = `"def456"`
	testClusterID    = "my-cluster"
)

// setupMockedProcessGroupingClient builds a client backed by a mock core.Client.
// params:       expected query params
// extraHeaders: additional headers expected to be set (e.g. If-None-Match)
// responseHeaders, responseBody, execErr: what ExecuteWriter returns
func setupMockedProcessGroupingClient(
	t *testing.T,
	params map[string]string,
	extraHeaders map[string]string,
	responseHeaders http.Header,
	responseBody []byte,
	execErr error,
) *ClientImpl {
	t.Helper()

	req := coremock.NewRequest(t)
	req.EXPECT().
		WithQueryParams(params).
		Return(req).Once()
	req.EXPECT().
		WithHeader("Accept", "application/cbor").
		Return(req).Once()

	for key, value := range extraHeaders {
		k, v := key, value
		req.EXPECT().
			WithHeader(k, v).
			Return(req).Once()
	}

	req.EXPECT().
		ExecuteWriter(mock.Anything).
		Run(func(writer io.Writer) {
			if execErr == nil && len(responseBody) > 0 {
				_, _ = writer.Write(responseBody)
			}
		}).
		Return(responseHeaders, execErr).Once()

	coreClient := coremock.NewClient(t)
	coreClient.EXPECT().
		GET(anyCtx, processGroupingConfigPath).
		Return(req).Once()

	return NewClient(coreClient, "", "")
}

func TestGetProcessGroupingConfig(t *testing.T) {
	t.Run("success_200_with_etag", func(t *testing.T) {
		respHeaders := http.Header{"Etag": []string{testResponseETag}}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{"kubernetesClusterId": testClusterID},
			map[string]string{"If-None-Match": testETag},
			respHeaders,
			[]byte(testCBORData),
			nil,
		)

		pgc, err := client.GetProcessGroupingConfig(t.Context(), testClusterID, testETag)
		require.NoError(t, err)
		assert.Equal(t, testCBORData, string(pgc.Data))
		// On 200, the returned ETag comes from the response header, not the input ETag
		assert.Equal(t, testResponseETag, pgc.ETag)
		assert.NotEqual(t, testETag, pgc.ETag)
	})

	t.Run("success_200_without_etag", func(t *testing.T) {
		respHeaders := http.Header{"Etag": []string{testResponseETag}}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{"kubernetesClusterId": testClusterID},
			nil,
			respHeaders,
			[]byte(testCBORData),
			nil,
		)

		pgc, err := client.GetProcessGroupingConfig(t.Context(), testClusterID, "")
		require.NoError(t, err)
		assert.Equal(t, testCBORData, string(pgc.Data))
		assert.Equal(t, testResponseETag, pgc.ETag)
	})

	t.Run("not_modified_304", func(t *testing.T) {
		httpErr := &core.HTTPError{StatusCode: 304}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{"kubernetesClusterId": testClusterID},
			map[string]string{"If-None-Match": testETag},
			nil,
			nil,
			httpErr,
		)

		pgc, err := client.GetProcessGroupingConfig(t.Context(), testClusterID, testETag)
		require.NoError(t, err)
		assert.Empty(t, string(pgc.Data))
		// On 304, the original ETag is returned for convenience
		assert.Equal(t, testETag, pgc.ETag)
	})

	t.Run("with_kubernetes_cluster_id", func(t *testing.T) {
		respHeaders := http.Header{"Etag": []string{testResponseETag}}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{"kubernetesClusterId": testClusterID},
			nil,
			respHeaders,
			[]byte(testCBORData),
			nil,
		)

		pgc, err := client.GetProcessGroupingConfig(t.Context(), testClusterID, "")
		require.NoError(t, err)
		assert.Equal(t, testResponseETag, pgc.ETag)
	})

	t.Run("empty_kubernetes_cluster_id_returns_error", func(t *testing.T) {
		// Create a client directly without mock setup since we expect early return
		coreClient := coremock.NewClient(t)
		client := NewClient(coreClient, "", "")

		pgc, err := client.GetProcessGroupingConfig(t.Context(), "", "")
		require.Error(t, err)
		assert.Nil(t, pgc)
	})

	t.Run("server_error", func(t *testing.T) {
		serverErr := &core.HTTPError{StatusCode: http.StatusInternalServerError, Message: "internal server error"}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{"kubernetesClusterId": testClusterID},
			nil,
			nil,
			nil,
			serverErr,
		)

		pgc, err := client.GetProcessGroupingConfig(t.Context(), testClusterID, "")
		require.Error(t, err)
		require.True(t, core.HasStatusCode(err, http.StatusInternalServerError))
		assert.Nil(t, pgc)
	})

	t.Run("bad_request", func(t *testing.T) {
		const badETag = "bad_etag"
		serverErr := &core.HTTPError{StatusCode: http.StatusBadRequest, Message: "bad request"}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{"kubernetesClusterId": testClusterID},
			map[string]string{"If-None-Match": badETag},
			nil,
			nil,
			serverErr,
		)

		pgc, err := client.GetProcessGroupingConfig(t.Context(), testClusterID, badETag)
		require.Error(t, err)
		require.True(t, core.HasStatusCode(err, http.StatusBadRequest))
		assert.Nil(t, pgc)
	})

	t.Run("not_found_404_endpoint_unavailable", func(t *testing.T) {
		httpErr := &core.HTTPError{StatusCode: http.StatusNotFound, Message: "endpoint not available"}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{"kubernetesClusterId": testClusterID},
			nil,
			nil,
			nil,
			httpErr,
		)

		pgc, err := client.GetProcessGroupingConfig(t.Context(), testClusterID, "")
		require.NoError(t, err)
		assert.NotNil(t, pgc)
		assert.Empty(t, pgc.Data)
		assert.Empty(t, pgc.ETag)
	})
}
