package dtclient

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

const (
	activeGateType = "ENVIRONMENT"
)

type ActiveGateAuthTokenInfo struct {
	TokenId string `json:"id"`
	Token   string `json:"token"`
}

type ActiveGateAuthTokenParams struct {
	Name           string `json:"name"`
	SeedToken      bool   `json:"seedToken"`
	ActiveGateType string `json:"activeGateType"`
}

func (dtc *dynatraceClient) GetActiveGateAuthToken(dynakubeName string) (*ActiveGateAuthTokenInfo, error) {
	body := &ActiveGateAuthTokenParams{
		Name:           dynakubeName,
		SeedToken:      false,
		ActiveGateType: activeGateType,
	}
	bodyData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	request, err := createBaseRequest(
		dtc.getActiveGateAuthTokenUrl(),
		http.MethodPost,
		dtc.apiToken,
		bytes.NewReader(bodyData),
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	response, err := dtc.httpClient.Do(request)
	if err != nil {
		log.Info("failed to retrieve ActiveGateAuthToken")
		return nil, err
	}

	defer func() {
		err := response.Body.Close()
		if err != nil {
			log.Error(err, err.Error())
		}
	}()

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		return nil, dtc.handleErrorResponseFromAPI(data, response.StatusCode)
	}

	authTokenInfo, err := dtc.readResponseForActiveGateAuthToken(data)
	if err != nil {
		return nil, err
	}

	return authTokenInfo, nil
}

func (dtc *dynatraceClient) readResponseForActiveGateAuthToken(response []byte) (*ActiveGateAuthTokenInfo, error) {
	agAuthToken := &ActiveGateAuthTokenInfo{}
	err := json.Unmarshal(response, agAuthToken)
	if err != nil {
		log.Error(err, "error unmarshalling ActiveGateAuthTokenInfo", "response", string(response))
		return nil, err
	}

	return agAuthToken, nil
}
