package dynatrace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

const APITokenHeader = "Api-Token "

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
	dynatraceAPIToken tokenType = iota
	dynatracePaaSToken
	installerURLToken     // in this case we don't care about the token
	defaultMaxResponseLen = 1000
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
	case dynatraceAPIToken:
		if dtc.apiToken == "" {
			return nil, errors.Errorf("not able to set token since api token is empty for request: %s", url)
		}

		authHeader = APITokenHeader + dtc.apiToken
	case dynatracePaaSToken:
		if dtc.paasToken == "" {
			return nil, errors.Errorf("not able to set token since paas token is empty for request: %s", url)
		}

		authHeader = APITokenHeader + dtc.paasToken
	case installerURLToken:
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
	req.Header.Add("Authorization", APITokenHeader+apiToken)

	if method == http.MethodPost {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

func (dtc *dynatraceClient) getServerResponseData(response *http.Response) ([]byte, error) {
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "error reading response body")
	}

	if response.StatusCode != http.StatusOK &&
		response.StatusCode != http.StatusCreated {
		return responseData, dtc.handleErrorResponseFromAPI(responseData, response.StatusCode, response.Header)
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

	hash := sha256.New()
	_, err = io.Copy(writer, io.TeeReader(resp.Body, hash))

	return hex.EncodeToString(hash.Sum(nil)), err
}

func (dtc *dynatraceClient) handleErrorResponseFromAPI(response []byte, statusCode int, headers http.Header) error {
	se := serverErrorResponse{}

	contentType := "unknown"
	proxy := ""

	if headers != nil {
		for _, field := range []string{"X-Forwarded-For", "Forwarded", "Via"} {
			if value := headers.Get(field); value != "" {
				proxy = value

				break
			}
		}

		if value := headers.Get("Content-Type"); value != "" {
			contentType = value
		}
	}

	if err := json.Unmarshal(response, &se); err != nil {
		var sb strings.Builder

		sb.WriteString(fmt.Sprintf("server returned status code %d", statusCode))

		if proxy != "" {
			sb.WriteString(fmt.Sprintf(" (via proxy %s)", proxy))
		}

		responseLen := min(getMaxResponseLen(), len(response))
		sb.WriteString(fmt.Sprintf("; can't unmarshal response (content-type: %s): %s", contentType, response[:responseLen]))

		return errors.New(sb.String())
	}

	return se.ErrorMessage
}

func getMaxResponseLen() int {
	if envVar, exists := os.LookupEnv("DT_CLIENT_API_ERROR_LOG_LEN"); exists {
		maxResponseLen, err := strconv.Atoi(envVar)
		if err == nil && maxResponseLen > 0 {
			return maxResponseLen
		}
	}

	return defaultMaxResponseLen
}
