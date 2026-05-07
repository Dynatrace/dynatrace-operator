package oneagent

import (
	"bytes"
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

const (
	processGroupingConfigPath    = "/v1/deployment/installer/agent/processgroupingconfig"
	parameterKubernetesClusterID = "kubernetesClusterId"
	requestHeaderEtag            = "If-None-Match"
	responseHeaderEtag           = "ETag"
)

type ProcessGroupConfig struct {
	ETag string
	Data []byte
}

// GetProcessGroupingConfig fetches the process grouping configuration as a binary CBOR stream.
//
// Parameters:
//   - kubernetesClusterId: optional Kubernetes cluster ID to scope the config. Empty string omits the parameter.
//   - etag: optional ETag from a previous response. When non-empty, sent as If-None-Match header.
//     If the server responds with 304 Not Modified, the underlying HTTP error is returned.
//     Use core.HasStatusCode(err, http.StatusNotModified) to check for this case.
//
// Returns:
//   - On HTTP 200: *ProcessGroupConfig with ETag from response header and CBOR data.
//   - On HTTP 304: *ProcessGroupConfig with the original ETag and nil Data, plus a non-nil error satisfying core.HasStatusCode(err, http.StatusNotModified).
//   - On other errors: non-nil error.
func (c *ClientImpl) GetProcessGroupingConfig(ctx context.Context, kubernetesClusterID string, etag string) (*ProcessGroupConfig, error) {
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

	var buf bytes.Buffer

	headers, err := req.ExecuteWriter(&buf)
	if core.HasStatusCode(err, http.StatusNotModified) {
		return &ProcessGroupConfig{ETag: etag}, err
	}

	if err != nil {
		return &ProcessGroupConfig{}, err
	}

	pgc := &ProcessGroupConfig{
		ETag: headers.Get(responseHeaderEtag),
		Data: buf.Bytes(),
	}

	return pgc, nil
}
