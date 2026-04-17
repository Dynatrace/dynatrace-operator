package oneagent

import (
	"bytes"
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
	testCBORData        = "\x81\x01" // minimal CBOR: array of one integer
	testETag            = `"abc123"`
	testResponseETag    = "def456"
	testResponseETagRaw = `"` + testResponseETag + `"`
	testClusterID       = "my-cluster"
)

// setupMockedProcessGroupingClient builds a Client backed by a mock core.APIClient.
// params:       expected query params
// extraHeaders: additional headers expected to be set (e.g. If-None-Match)
// responseHeaders, responseBody, execErr: what ExecuteWriterWithHeaders returns
func setupMockedProcessGroupingClient(
	t *testing.T,
	params map[string]string,
	extraHeaders map[string]string,
	responseHeaders http.Header,
	responseBody []byte,
	execErr error,
) *Client {
	t.Helper()

	req := coremock.NewAPIRequest(t)
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
		ExecuteWriterWithHeaders(mock.Anything).
		Run(func(writer io.Writer) {
			if execErr == nil && len(responseBody) > 0 {
				_, _ = writer.Write(responseBody)
			}
		}).
		Return(responseHeaders, execErr).Once()

	apiClient := coremock.NewAPIClient(t)
	apiClient.EXPECT().
		GET(t.Context(), processGroupingConfigPath).
		Return(req).Once()

	return NewClient(apiClient, "", "")
}

func TestGetProcessGroupingConfig(t *testing.T) {
	t.Run("success_200_with_etag", func(t *testing.T) {
		var buf bytes.Buffer
		respHeaders := http.Header{"Etag": []string{testResponseETagRaw}}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{},
			map[string]string{"If-None-Match": testETag},
			respHeaders,
			[]byte(testCBORData),
			nil,
		)

		returnedETag, err := client.GetProcessGroupingConfig(t.Context(), "", testETag, &buf)
		require.NoError(t, err)
		assert.Equal(t, testCBORData, buf.String())
		// On 200, the returned ETag comes from the response header, not the input ETag
		assert.Equal(t, testResponseETag, returnedETag)
		assert.NotEqual(t, testETag, returnedETag)
	})

	t.Run("success_200_without_etag", func(t *testing.T) {
		var buf bytes.Buffer
		respHeaders := http.Header{"Etag": []string{testResponseETagRaw}}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{},
			nil,
			respHeaders,
			[]byte(testCBORData),
			nil,
		)

		returnedETag, err := client.GetProcessGroupingConfig(t.Context(), "", "", &buf)
		require.NoError(t, err)
		assert.Equal(t, testCBORData, buf.String())
		assert.Equal(t, testResponseETag, returnedETag)
	})

	t.Run("not_modified_304", func(t *testing.T) {
		var buf bytes.Buffer
		httpErr := &core.HTTPError{StatusCode: 304}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{},
			map[string]string{"If-None-Match": testETag},
			nil,
			nil,
			httpErr,
		)

		returnedETag, err := client.GetProcessGroupingConfig(t.Context(), "", testETag, &buf)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotModified, "expected ErrNotModified, got %v", err)
		assert.Empty(t, buf.String())
		// On 304, the original ETag is returned for convenience
		assert.Equal(t, testETag, returnedETag)
	})

	t.Run("with_kubernetes_cluster_id", func(t *testing.T) {
		var buf bytes.Buffer
		respHeaders := http.Header{"Etag": []string{testResponseETagRaw}}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{"kubernetesClusterId": testClusterID},
			nil,
			respHeaders,
			[]byte(testCBORData),
			nil,
		)

		returnedETag, err := client.GetProcessGroupingConfig(t.Context(), testClusterID, "", &buf)
		require.NoError(t, err)
		assert.Equal(t, testResponseETag, returnedETag)
	})

	t.Run("without_kubernetes_cluster_id", func(t *testing.T) {
		var buf bytes.Buffer
		respHeaders := http.Header{"Etag": []string{testResponseETagRaw}}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{},
			nil,
			respHeaders,
			[]byte(testCBORData),
			nil,
		)

		returnedETag, err := client.GetProcessGroupingConfig(t.Context(), "", "", &buf)
		require.NoError(t, err)
		assert.Equal(t, testResponseETag, returnedETag)
	})

	t.Run("server_error", func(t *testing.T) {
		var buf bytes.Buffer
		serverErr := &core.HTTPError{StatusCode: 500, Message: "internal server error"}

		client := setupMockedProcessGroupingClient(t,
			map[string]string{},
			nil,
			nil,
			nil,
			serverErr,
		)

		returnedETag, err := client.GetProcessGroupingConfig(t.Context(), "", "", &buf)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrNotModified)
		assert.Empty(t, returnedETag)
		assert.Empty(t, buf.String())
	})
}
