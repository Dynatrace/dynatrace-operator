package dtclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type hostInfo struct {
	version  string
	entityID string
}

// client implements the Client interface.
type dynatraceClient struct {
	url       string
	apiToken  string
	paasToken string

	httpClient *http.Client

	hostCache map[string]hostInfo
}

type tokenType int

const (
	dynatraceApiToken tokenType = iota
	dynatracePaaSToken
)

var logger = log.Log.WithName("dynatrace.client")

// makeRequest does an HTTP request by formatting the URL from the given arguments and returns the response.
// The response body must be closed by the caller when no longer used.
func (dc *dynatraceClient) makeRequest(url string, tokenType tokenType) (*http.Response, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error initialising http request: %s", err.Error())
	}

	var authHeader string

	if dc.paasToken == "" || dc.apiToken == "" {
		return nil, errors.New("Not able to set token since token is empty!")
	}

	switch tokenType {
	case dynatraceApiToken:
		authHeader = fmt.Sprintf("Api-Token %s", dc.apiToken)
	case dynatracePaaSToken:
		authHeader = fmt.Sprintf("Api-Token %s", dc.paasToken)
	default:
		return nil, errors.New("Unable to determine token to set in headers")
	}

	req.Header.Add("Authorization", authHeader)

	return dc.httpClient.Do(req)
}

func (dc *dynatraceClient) getServerResponseData(response *http.Response) ([]byte, error) {
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return responseData, dc.handleErrorResponseFromAPI(responseData, response.StatusCode)
	}

	return responseData, nil
}

func (dc *dynatraceClient) handleErrorResponseFromAPI(response []byte, statusCode int) error {
	se := serverErrorResponse{}
	if err := json.Unmarshal(response, &se); err != nil {
		return fmt.Errorf("response error: %d, can't unmarshal json response: %w", statusCode, err)
	}

	return se.ErrorMessage
}

func (dc *dynatraceClient) getHostInfoForIP(ip string) (*hostInfo, error) {
	if len(dc.hostCache) == 0 {
		err := dc.buildHostCache()
		if err != nil {
			return nil, fmt.Errorf("error building hostcache from dynatrace cluster: %w", err)
		}
	}

	switch hostInfo, ok := dc.hostCache[ip]; {
	case !ok:
		return nil, errors.New("host not found")
	default:
		return &hostInfo, nil
	}
}

func (dc *dynatraceClient) buildHostCache() error {
	var url string = fmt.Sprintf("%s/v1/entity/infrastructure/hosts?includeDetails=false", dc.url)
	resp, err := dc.makeRequest(url, dynatraceApiToken)
	if err != nil {
		return err
	}
	defer func() {
		// Swallow error
		_ = resp.Body.Close()
	}()

	responseData, err := dc.getServerResponseData(resp)
	if err != nil {
		return err
	}

	err = dc.setHostCacheFromResponse(responseData)
	if err != nil {
		return err
	}

	return nil
}

func (dc *dynatraceClient) setHostCacheFromResponse(response []byte) error {
	type hostInfoResponse struct {
		IPAddresses  []string
		AgentVersion *struct {
			Major     int
			Minor     int
			Revision  int
			Timestamp string
		}
		EntityID string
	}

	dc.hostCache = make(map[string]hostInfo)

	hostInfoResponses := []hostInfoResponse{}
	err := json.Unmarshal(response, &hostInfoResponses)
	if err != nil {
		logger.Error(err, "error unmarshalling json response")
		return err
	}

	for _, info := range hostInfoResponses {
		hostInfo := hostInfo{entityID: info.EntityID}

		v := info.AgentVersion
		if v == nil {
			continue
		}

		hostInfo.version = fmt.Sprintf("%d.%d.%d.%s", v.Major, v.Minor, v.Revision, v.Timestamp)
		for _, ip := range info.IPAddresses {
			dc.hostCache[ip] = hostInfo
		}
	}

	return nil
}

type serverErrorResponse struct {
	ErrorMessage ServerError `json:"error"`
}

// ServerError represents an error returned from the server (e.g. authentication failure).
type ServerError struct {
	Code    int
	Message string
}

// Error formats the server error code and message.
func (e ServerError) Error() string {
	if len(e.Message) == 0 && e.Code == 0 {
		return "unknown server error"
	}

	return fmt.Sprintf("dynatrace server error %d: %s", int64(e.Code), e.Message)
}
