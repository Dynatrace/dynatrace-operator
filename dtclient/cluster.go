package dtclient

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

const (
	clusterVersionEndpoint = "/v1/config/clusterversion"
)

type ClusterInfo struct {
	Version string `json:"version"`
}

func (dtc *dynatraceClient) GetClusterInfo() (*ClusterInfo, error) {
	result := ClusterInfo{}
	url := fmt.Sprintf("%s%s", dtc.url, clusterVersionEndpoint)
	resp, err := dtc.makeRequest(url, dynatraceApiToken)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer func() {
		// Unable to do anything, swallow error
		_ = resp.Body.Close()
	}()

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err = json.Unmarshal(responseData, &result); err != nil {
		return nil, errors.WithStack(err)
	}

	return &result, nil
}
