package dtclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	activeGateType    = "ENVIRONMENT"
	authTokenValidity = time.Hour * 24 * 60
)

type ActiveGateAuthTokenInfo struct {
	TokenId string `json:"id"`
	Token   string `json:"token"`
}

type ActiveGateAuthTokenParams struct {
	Name           string `json:"name"`
	SeedToken      bool   `json:"seedToken"`
	ActiveGateType string `json:"activeGateType"`
	ExpirationDate string `json:"expirationDate"`
}

func (dtc *dynatraceClient) GetActiveGateAuthToken(dynakubeName string) (*ActiveGateAuthTokenInfo, error) {
	request, cancel, err := dtc.createAuthTokenRequest(dynakubeName)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer cancel()

	response, err := dtc.httpClient.Do(request)

	if err != nil {
		log.Info("failed to retrieve ag-auth-token")
		return nil, err
	}

	defer CloseBodyAfterRequest(response)

	authTokenInfo, err := dtc.handleAuthTokenResponse(response)
	if err != nil {
		log.Info("failed to handle ag-auth-token response")
		return nil, err
	}

	return authTokenInfo, nil
}

func (dtc *dynatraceClient) createAuthTokenRequest(dynakubeName string) (*http.Request, context.CancelFunc, error) {
	body := &ActiveGateAuthTokenParams{
		Name:           dynakubeName,
		SeedToken:      false,
		ActiveGateType: activeGateType,
		ExpirationDate: getAuthTokenExpirationDate(),
	}
	bodyData, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}

	request, cancel, err := createBaseRequest(
		dtc.getActiveGateAuthTokenUrl(),
		http.MethodPost,
		dtc.apiToken,
		bytes.NewReader(bodyData),
	)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return request, cancel, nil
}

func (dtc *dynatraceClient) handleAuthTokenResponse(response *http.Response) (*ActiveGateAuthTokenInfo, error) {
	data, err := dtc.getServerResponseData(response)
	if err != nil {
		return nil, dtc.handleErrorResponseFromAPI(data, response.StatusCode)
	}

	authTokenInfo, err := dtc.readResponseForActiveGateAuthToken(data)
	if err != nil {
		return nil, err
	}

	return authTokenInfo, err
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

func getAuthTokenExpirationDate() string {
	return time.Now().Add(authTokenValidity).UTC().Format(time.RFC3339)
}
