package dtclient

import (
	"encoding/json"
	"fmt"
)

const (
	clusterVersionEndpoint = "/v1/config/clusterversion"
)

type ClusterInfo struct {
	Version string `json:"version"`
}

func (dc *dynatraceClient) GetClusterInfo() (*ClusterInfo, error) {
	result := ClusterInfo{}
	url := fmt.Sprintf("%s%s", dc.url, clusterVersionEndpoint)
	resp, err := dc.makeRequest(url, dynatraceApiToken)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Unable to do anything, swallow error
		_ = resp.Body.Close()
	}()

	responseData, err := dc.getServerResponseData(resp)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(responseData, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
