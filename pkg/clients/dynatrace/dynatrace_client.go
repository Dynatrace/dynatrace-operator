package dynatrace

import (
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

const ApiTokenHeader = "Api-Token "

// client implements the Client interface.
type dynatraceClient struct {

	// Set for testing purposes, leave the default zero value to use the current time.
	now time.Time

	httpClient *http.Client

	url       string
	apiToken  string
	paasToken string

	networkZone string

	hostGroup string
}

type tokenType int

const (
	dynatraceApiToken tokenType = iota
	dynatracePaaSToken
	installerUrlToken // in this case we don't care about the token
)

// makeRequest does an HTTP request by formatting the URL from the given arguments and returns the response.
// The response body must be closed by the caller when no longer used.
func (dtc *dynatraceClient) makeRequest(ctx context.Context, url string, tokenType tokenType) (*http.Response, error) {
	// TODO: introduce ctx into dynatrace client
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "error initializing http request")
	}

	var authHeader string

	switch tokenType {
	case dynatraceApiToken:
		if dtc.apiToken == "" {
			return nil, errors.Errorf("not able to set token since api token is empty for request: %s", url)
		}

		authHeader = ApiTokenHeader + dtc.apiToken
	case dynatracePaaSToken:
		if dtc.paasToken == "" {
			return nil, errors.Errorf("not able to set token since paas token is empty for request: %s", url)
		}

		authHeader = ApiTokenHeader + dtc.paasToken
	case installerUrlToken:
		return dtc.httpClient.Do(req)
	default:
		return nil, errors.Errorf("unknown token type (%d), unable to determine token to set in headers", tokenType)
	}

	req.Header.Add("Authorization", authHeader)

	return dtc.httpClient.Do(req)
}

func createBaseRequest(ctx context.Context, url, method, apiToken string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, errors.WithMessage(err, "error initializing http request")
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", ApiTokenHeader+apiToken)

	if method == http.MethodPost {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

func (dtc *dynatraceClient) getServerResponseData(response *http.Response) ([]byte, error) {
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "error reading response")
	}

	if response.StatusCode != http.StatusOK &&
		response.StatusCode != http.StatusCreated {
		return responseData, dtc.handleErrorResponseFromAPI(responseData, response.StatusCode)
	}

	return responseData, nil
}

func (dtc *dynatraceClient) makeRequestAndUnmarshal(ctx context.Context, url string, token tokenType, response interface{}) error {
	resp, err := dtc.makeRequest(ctx, url, token)
	if err != nil {
		return err
	}

	defer utils.CloseBodyAfterRequest(resp)

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return err
	}

	return json.Unmarshal(responseData, &response)
}

func (dtc *dynatraceClient) makeRequestForBinary(ctx context.Context, url string, token tokenType, writer io.Writer) (string, error) {
	resp, err := dtc.makeRequest(ctx, url, token)
	if err != nil {
		return "", err
	}

	defer utils.CloseBodyAfterRequest(resp)

	if resp.StatusCode != http.StatusOK {
		var errorResponse serverErrorResponse

		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			return "", err
		}

		return "", errors.Errorf("dynatrace server error %d: %s", errorResponse.ErrorMessage.Code, errorResponse.ErrorMessage.Message)
	}

	hash := md5.New() //nolint:gosec
	_, err = io.Copy(writer, io.TeeReader(resp.Body, hash))

	return hex.EncodeToString(hash.Sum(nil)), err
}

func (dtc *dynatraceClient) handleErrorResponseFromAPI(response []byte, statusCode int) error {
	se := serverErrorResponse{}
	if err := json.Unmarshal(response, &se); err != nil {
		return errors.WithMessagef(err, "response error, can't unmarshal json response %d", statusCode)
	}

	return se.ErrorMessage
}
