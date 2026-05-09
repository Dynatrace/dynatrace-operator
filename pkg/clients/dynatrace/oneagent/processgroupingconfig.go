package oneagent

import (
	"context"
	"io"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

const (
	processGroupingConfigPath    = "/v1/deployment/installer/agent/processgroupingconfig"
	parameterKubernetesClusterID = "kubernetesClusterId"
	requestHeaderEtag            = "If-None-Match"
	responseHeaderEtag           = "ETag"
)

// GetProcessGroupingConfig fetches the process grouping configuration as a binary CBOR stream.
//
// Parameters:
//   - kubernetesClusterId: optional Kubernetes cluster ID to scope the config. Empty string omits the parameter.
//   - etag: optional ETag from a previous response. When non-empty, sent as If-None-Match header.
//     If the server responds with 304 Not Modified, the underlying HTTP error is returned.
//     Use core.HasStatusCode(err, http.StatusNotModified) to check for this case.
//   - writer: the io.Writer to stream the CBOR response body into.
//
// Returns:
//   - The ETag value from the response header on success (HTTP 200), or the original etag on 304.
//   - An error if the request failed. On HTTP 304, the error satisfies core.HasStatusCode(err, http.StatusNotModified).
func (c *ClientImpl) GetProcessGroupingConfig(ctx context.Context, kubernetesClusterID string, etag string, writer io.Writer) (string, error) {
	params := map[string]string{}
	if kubernetesClusterID != "" {
		params[parameterKubernetesClusterID] = kubernetesClusterID
	}

	req := c.apiClient.GET(ctx, processGroupingConfigPath).
		WithQueryParams(params).
		WithHeader("Accept", "application/cbor")

	if etag != "" {
		req = req.WithHeader(requestHeaderEtag, etag)
	}

	headers, err := req.ExecuteWriter(writer)
	if core.HasStatusCode(err, http.StatusNotModified) {
		return etag, err
	}

	if err != nil {
		return "", err
	}

	return headers.Get(responseHeaderEtag), nil
}
