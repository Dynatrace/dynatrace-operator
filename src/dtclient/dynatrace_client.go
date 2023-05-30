package dtclient

import (
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
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

	networkZone string

	disableHostsRequests bool

	httpClient *http.Client

	hostCache map[string]hostInfo

	// Set for testing purposes, leave the default zero value to use the current time.
	now time.Time
}

type tokenType int

const (
	dynatraceApiToken tokenType = iota
	dynatracePaaSToken
	installerUrlToken // in this case we don't care about the token
)

var defaultConnectionTimeout = 15 * time.Minute

// makeRequest does an HTTP request by formatting the URL from the given arguments and returns the response.
// The response body must be closed by the caller when no longer used.
func (dtc *dynatraceClient) makeRequest(url string, tokenType tokenType) (*http.Response, error) {
	req, err := createRequestWithDefaultTimeout(url, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	var authHeader string

	switch tokenType {
	case dynatraceApiToken:
		if dtc.apiToken == "" {
			return nil, errors.Errorf("not able to set token since api token is empty for request: %s", url)
		}
		authHeader = fmt.Sprintf("Api-Token %s", dtc.apiToken)
	case dynatracePaaSToken:
		if dtc.paasToken == "" {
			return nil, errors.Errorf("not able to set token since paas token is empty for request: %s", url)
		}
		authHeader = fmt.Sprintf("Api-Token %s", dtc.paasToken)
	case installerUrlToken:
		return dtc.httpClient.Do(req)
	default:
		return nil, errors.New("unable to determine token to set in headers")
	}

	req.Header.Add("Authorization", authHeader)

	return dtc.httpClient.Do(req)
}

func createBaseRequest(url, method, apiToken string, body io.Reader) (*http.Request, error) {
	req, err := createRequestWithDefaultTimeout(url, method, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", apiToken))

	if method == http.MethodPost {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

func createRequestWithDefaultTimeout(url, method string, body io.Reader) (*http.Request, error) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(defaultConnectionTimeout))

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, errors.WithMessage(err, "error initializing http request")
	}

	return req, nil
}

func (dtc *dynatraceClient) getServerResponseData(response *http.Response) ([]byte, error) {
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "error reading response")
	}

	if response.StatusCode != http.StatusOK &&
		response.StatusCode != http.StatusCreated {
		return responseData, dtc.handleErrorResponseFromAPI(responseData, response.StatusCode)
	}

	return responseData, nil
}

func (dtc *dynatraceClient) makeRequestAndUnmarshal(url string, token tokenType, response interface{}) error {
	resp, err := dtc.makeRequest(url, token)
	if err != nil {
		return err
	}
	defer CloseBodyAfterRequest(resp)

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return err
	}

	return json.Unmarshal(responseData, &response)
}

func (dtc *dynatraceClient) makeRequestForBinary(url string, token tokenType, writer io.Writer) (string, error) {
	resp, err := dtc.makeRequest(url, token)
	if err != nil {
		return "", err
	}
	defer CloseBodyAfterRequest(resp)

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

func (dtc *dynatraceClient) getHostInfoForIP(ip string) (*hostInfo, error) {
	if len(dtc.hostCache) == 0 {
		err := dtc.buildHostCache()
		if err != nil {
			return nil, errors.WithMessage(err, "error building host-cache from dynatrace cluster")
		}
	}

	switch hostInfo, ok := dtc.hostCache[ip]; {
	case !ok:
		return nil, errors.New("host not found")
	default:
		return &hostInfo, nil
	}
}

func (dtc *dynatraceClient) buildHostCache() error {
	if dtc.disableHostsRequests {
		return nil
	}

	resp, err := dtc.makeRequest(dtc.getHostsUrl(), dynatraceApiToken)
	if err != nil {
		return errors.WithStack(err)
	}
	defer CloseBodyAfterRequest(resp)

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return errors.WithStack(err)
	}

	err = dtc.setHostCacheFromResponse(responseData)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type hostInfoResponse struct {
	IPAddresses  []string
	AgentVersion *struct {
		Major     int
		Minor     int
		Revision  int
		Timestamp string
	}
	EntityID          string
	NetworkZoneID     string
	LastSeenTimestamp int64
}

func (dtc *dynatraceClient) setHostCacheFromResponse(response []byte) error {
	dtc.hostCache = make(map[string]hostInfo)

	hostInfoResponses, err := dtc.extractHostInfoResponse(response)
	if err != nil {
		return err
	}

	now := dtc.now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var inactive []string
	for _, info := range hostInfoResponses {
		// If we haven't seen this host in the last 30 minutes, ignore it.
		if tm := time.Unix(info.LastSeenTimestamp/1000, 0).UTC(); tm.Before(now.Add(-30 * time.Minute)) {
			inactive = append(inactive, info.EntityID)
			continue
		}

		nz := info.NetworkZoneID

		if (dtc.networkZone != "" && nz == dtc.networkZone) || (dtc.networkZone == "" && (nz == "default" || nz == "")) {
			hostInfo := hostInfo{entityID: info.EntityID}

			if v := info.AgentVersion; v != nil {
				hostInfo.version = fmt.Sprintf("%d.%d.%d.%s", v.Major, v.Minor, v.Revision, v.Timestamp)
			}

			dtc.updateHostCache(info, hostInfo)
		}
	}

	if len(inactive) > 0 {
		log.Info("hosts cache: ignoring inactive hosts", "ids", inactive)
	}

	return nil
}

func (dtc *dynatraceClient) updateHostCache(info hostInfoResponse, hostInfo hostInfo) {
	for _, ip := range info.IPAddresses {
		if old, ok := dtc.hostCache[ip]; ok {
			log.Info("hosts cache: replacing host", "ip", ip, "new", hostInfo.entityID, "old", old.entityID)
		}

		dtc.hostCache[ip] = hostInfo
	}
}

func (dtc *dynatraceClient) extractHostInfoResponse(response []byte) ([]hostInfoResponse, error) {
	var hostInfoResponses []hostInfoResponse
	err := json.Unmarshal(response, &hostInfoResponses)
	if err != nil {
		log.Error(err, "error unmarshalling json response", "response", string(response))
		return nil, errors.WithStack(err)
	}
	return hostInfoResponses, nil
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
