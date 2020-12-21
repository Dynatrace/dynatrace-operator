package dtclient

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"net/url"
)

type ActiveGateQuery struct {
	Hostname       string
	NetworkAddress string
	NetworkZone    string
	UpdateStatus   string
}

type ActiveGate struct {
	NetworkAddresses []string
	AutoUpdateStatus string
	OfflineSince     int64
	Version          string
	Hostname         string
	NetworkZone      string
}

func (dtc *dynatraceClient) QueryActiveGates(query *ActiveGateQuery) ([]ActiveGate, error) {
	activeGateURL := fmt.Sprintf("%s/v2/activeGates?%s", dtc.url, query.buildQueryParams())
	dtc.logger.Info("querying activegates", "activeGateURL", activeGateURL)
	response, err := dtc.makeRequest(activeGateURL, dynatraceApiToken)
	if err != nil {
		dtc.logger.Error(err, err.Error())
		return nil, errors.WithStack(err)
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			dtc.logger.Error(err, "error closing response body")
		}
	}()

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		dtc.logger.Error(err, err.Error())
		return nil, errors.WithStack(err)
	}

	var result []ActiveGate
	activegates, err := dtc.readResponseForActiveGates(data)
	if err != nil {
		dtc.logger.Error(err, err.Error())
		return nil, errors.WithStack(err)
	}

	for _, activegate := range activegates {
		if activegate.OfflineSince <= 0 {
			result = append(result, activegate)
		}
	}

	return result, nil
}

func (dtc *dynatraceClient) QueryOutdatedActiveGates(query *ActiveGateQuery) ([]ActiveGate, error) {
	query.UpdateStatus = StatusOutdated
	return dtc.QueryActiveGates(query)
}

func (query *ActiveGateQuery) buildQueryParams() string {
	values := url.Values{}
	if query != nil {
		if query.Hostname != "" {
			values.Set("hostname", query.Hostname)
		}
		if query.NetworkZone != "" {
			values.Set("networkZone", query.NetworkZone)
		}
		if query.NetworkAddress != "" {
			values.Set("networkAddress", query.NetworkAddress)
		}
		if query.UpdateStatus != "" {
			values.Set("updateStatus", query.UpdateStatus)
		}
	}

	values.Set("osType", OsLinux)
	values.Set("type", "ENVIRONMENT")

	return values.Encode()
}

func (dtc *dynatraceClient) readResponseForActiveGates(data []byte) ([]ActiveGate, error) {
	type jsonResponse struct {
		ActiveGates []ActiveGate
	}

	jr := &jsonResponse{}
	err := json.Unmarshal(data, jr)
	if err != nil {
		dtc.logger.Error(err, "cannot unmarshal activegate response")
		return nil, errors.WithStack(err)
	}
	return jr.ActiveGates, nil
}

const (
	OsLinux        = "LINUX"
	StatusOutdated = "OUTDATED"
)
