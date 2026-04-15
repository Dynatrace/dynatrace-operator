package oneagent

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

const (
	processGroupingConfigPath = "/v1/deployment/installer/agent/processgroupingconfig"
)

// ErrNotModified is returned when the server responds with 304 Not Modified,
// indicating the content has not changed since the ETag provided in the request.
var ErrNotModified = errors.New("server returned 304 Not Modified")

// GetProcessGroupingConfig fetches the process grouping configuration as a binary CBOR stream.
//
// Parameters:
//   - kubernetesClusterId: optional Kubernetes cluster ID to scope the config. Empty string omits the parameter.
//   - etag: optional ETag from a previous response. When non-empty, sent as If-None-Match header.
//     If the server responds with 304 Not Modified, ErrNotModified is returned.
//   - writer: the io.Writer to stream the CBOR response body into.
//
// Returns:
//   - The ETag value from the response header on success (HTTP 200), or the original etag on 304.
//   - An error if the request failed. Returns ErrNotModified on HTTP 304.
func (c *Client) GetProcessGroupingConfig(ctx context.Context, kubernetesClusterID string, etag string, writer io.Writer) (string, error) {
	params := map[string]string{}
	if kubernetesClusterID != "" {
		params["kubernetesClusterId"] = kubernetesClusterID
	}

	req := c.apiClient.GET(ctx, processGroupingConfigPath).
		WithQueryParams(params).
		WithPaasToken().
		WithHeader("Accept", "application/cbor")

	if etag != "" {
		req = req.WithHeader("If-None-Match", etag)
	}

	headers, err := req.ExecuteWriterWithHeaders(writer)
	if core.HasStatusCode(err, http.StatusNotModified) {
		return etag, ErrNotModified
	}

	if err != nil {
		return "", err
	}

	return headers.Get("ETag"), nil
}
