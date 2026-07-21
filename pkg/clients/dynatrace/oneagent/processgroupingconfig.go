package oneagent

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
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
//   - kubernetesClusterId: required Kubernetes cluster ID to scope the config. Empty string returns an error.
//   - etag: optional ETag from a previous response. When non-empty, sent as If-None-Match header.
//     If the server responds with 304 Not Modified, returns a ProcessGroupConfig with the original ETag.
//
// Returns:
//   - On HTTP 200: *ProcessGroupConfig with ETag from response header and CBOR data, nil error.
//   - On HTTP 304: *ProcessGroupConfig with the original ETag and nil Data, nil error.
//   - On HTTP 404: *ProcessGroupConfig (empty), nil error. Endpoint not available.
//   - On other errors: non-nil error.
func (c *ClientImpl) GetProcessGroupingConfig(ctx context.Context, kubernetesClusterID string, etag string) (*ProcessGroupConfig, error) {
	ctx, log := logd.NewFromContext(ctx, loggerName)

	if kubernetesClusterID == "" {
		return nil, errors.New("kubernetesClusterID is required")
	}

	params := map[string]string{
		parameterKubernetesClusterID: kubernetesClusterID,
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
		return &ProcessGroupConfig{ETag: etag}, nil
	}

	if err != nil {
		if core.IsNotFound(err) {
			log.Info("process grouping config not available on cluster, skipping getting process grouping config")

			return &ProcessGroupConfig{}, nil
		}

		return nil, err
	}

	pgc := &ProcessGroupConfig{
		ETag: headers.Get(responseHeaderEtag),
		Data: buf.Bytes(),
	}

	return pgc, nil
}
