package dynatrace

import (
	"context"
	"crypto/md5" //nolint:gosec
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

type HostNotFoundErr struct {
	IP string
}

func (e HostNotFoundErr) Error() string {
	return fmt.Sprintf("host not found for ip: %v", e.IP)
}

type hostInfo struct {
	version  string
	entityID string
}

// client implements the Client interface.
type dynatraceClient struct {

	// Set for testing purposes, leave the default zero value to use the current time.
	now time.Time

	httpClient *http.Client

	hostCache map[string]hostInfo

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
		return nil, errors.WithMessage(err, "error reading response")
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

	hash := md5.New() //nolint:gosec
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

		sb.WriteString(fmt.Sprintf("Server returned status code %d", statusCode))

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

func (dtc *dynatraceClient) getHostInfoForIP(ctx context.Context, ip string) (*hostInfo, error) {
	if len(dtc.hostCache) == 0 {
		err := dtc.buildHostCache(ctx)
		if err != nil {
			return nil, errors.WithMessage(err, "error building host-cache from dynatrace cluster")
		}
	}

	switch hostInfo, ok := dtc.hostCache[ip]; {
	case !ok:
		return nil, HostNotFoundErr{IP: ip}
	default:
		return &hostInfo, nil
	}
}

func (dtc *dynatraceClient) buildHostCache(ctx context.Context) error {
	resp, err := dtc.makeRequest(ctx, dtc.getHostsURL(), dynatraceAPIToken)
	if err != nil {
		return errors.WithStack(err)
	}

	defer utils.CloseBodyAfterRequest(resp)

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
	AgentVersion *struct {
		Timestamp string
		Major     int
		Minor     int
		Revision  int
	}
	EntityID          string
	NetworkZoneID     string
	IPAddresses       []string
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
