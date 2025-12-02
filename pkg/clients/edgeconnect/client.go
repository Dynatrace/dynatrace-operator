package edgeconnect

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	contentTypeJSON = "application/json"
)

type client struct {
	clientcredentials.Config
	ctx               context.Context
	httpClient        *http.Client
	baseURL           string
	customCA          []byte
	disableKeepAlives bool
}

// Option can be passed to NewClient and customizes the created client instance.
type Option func(client *client)

// NewClient creates a REST client for the given API base URL
// options can be used to customize the created client.
func NewClient(clientID, clientSecret string, options ...Option) (Client, error) {
	c := &client{
		Config: clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
	}

	for _, opt := range options {
		opt(c)
	}

	httpClient := c.Client(c.ctx)
	if httpClient == nil {
		return nil, errors.New("can't create http client for edge connect")
	}

	ot, ok := httpClient.Transport.(*oauth2.Transport)
	if !ok {
		return nil, errors.New("unexpected transport type")
	}
	if ot.Base == nil {
		ot.Base = &http.Transport{}
	}

	if c.disableKeepAlives {
		if t, ok := ot.Base.(*http.Transport); ok {
			t.DisableKeepAlives = true
		}
	}

	if c.customCA != nil {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			return nil, errors.Wrap(err, "read system certificates")
		}

		if ok := rootCAs.AppendCertsFromPEM(c.customCA); !ok {
			return nil, errors.New("append custom certs")
		}

		t := ot.Base.(*http.Transport)
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}

		t.TLSClientConfig.RootCAs = rootCAs
	}

	c.httpClient = httpClient

	return c, nil
}

func WithBaseURL(url string) func(*client) {
	return func(c *client) {
		c.baseURL = url
	}
}

func WithTokenURL(url string) func(*client) {
	return func(c *client) {
		c.TokenURL = url
	}
}

func WithOauthScopes(scopes []string) func(*client) {
	return func(c *client) {
		c.Scopes = scopes
	}
}

func WithCustomCA(caData []byte) func(*client) {
	return func(c *client) {
		c.customCA = caData
	}
}

// WithContext can set context for client
// NB: via context you can override default http client to add Proxy or CA certificates
func WithContext(ctx context.Context) func(*client) {
	return func(c *client) {
		c.ctx = ctx
	}
}

func WithKeepAlive(keepAlive bool) func(*client) {
	return func(c *client) {
		c.disableKeepAlives = !keepAlive
	}
}

// ServerError represents an error returned from the server (e.g. authentication failure).
type ServerError struct {
	Message string       `json:"message,omitempty"`
	Details DetailsError `json:"details"`
	Code    int          `json:"code,omitempty"`
}

// DetailsError represents details of errors
type DetailsError struct {
	ConstraintViolations []ConstraintViolations `json:"constraintViolations"`
	MissingScopes        []string               `json:"missingScopes,omitempty"`
}

// ConstraintViolations represents constraint violation errors
type ConstraintViolations struct {
	Message           string `json:"message"`
	Path              string `json:"path,omitempty"`
	ParameterLocation string `json:"parameterLocation,omitempty"`
}

// Error formats the server error code and message.
func (e ServerError) Error() string {
	if len(e.Message) == 0 && e.Code == 0 {
		return "unknown server error"
	}

	return fmt.Sprintf("edgeconnect server error %d: %s: details: %s", int64(e.Code), e.Message, e.Details)
}

type serverErrorResponse struct {
	ErrorMessage ServerError `json:"error"`
}

// Edgeconnect client handles two different APIs with different response schemas.
// That's why we need Settings API specific structs

// Settings API response
type SettingsAPIResponse struct {
	Error SettingsAPIError `json:"error"`
	Code  int              `json:"code"`
}

// Settings API error field
type SettingsAPIError struct {
	Message              string                 `json:"message,omitempty"`
	ConstraintViolations []ConstraintViolations `json:"constraintViolations"`
	Code                 int                    `json:"code,omitempty"`
}

func (e SettingsAPIError) Error() string {
	if len(e.Message) == 0 && e.Code == 0 {
		return "unknown server error"
	}

	return fmt.Sprintf("edgeconnect Settings API error: code=%d, message=%s, ConstraintViolations=%v", int64(e.Code), e.Message, e.ConstraintViolations)
}

func (c *client) handleErrorResponseFromAPI(response []byte, statusCode int) error {
	se := serverErrorResponse{}
	if err := json.Unmarshal(response, &se); err != nil {
		return errors.WithStack(errors.WithMessagef(err, "response error, can't unmarshal json response %d", statusCode))
	}

	return se.ErrorMessage
}

func (c *client) handleErrorResponseFromSettingsAPI(response []byte, statusCode int) error {
	se := SettingsAPIResponse{}
	if err := json.Unmarshal(response, &se); err != nil {
		return errors.WithStack(errors.WithMessagef(err, "response error, can't unmarshal json response %d", statusCode))
	}

	return se.Error
}

func (c *client) getServerResponseData(response *http.Response) ([]byte, error) {
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "error reading response")
	}

	if response.StatusCode != http.StatusOK &&
		response.StatusCode != http.StatusCreated {
		return responseData, c.handleErrorResponseFromAPI(responseData, response.StatusCode)
	}

	return responseData, nil
}

func (c *client) getSettingsAPIResponseData(response *http.Response) ([]byte, error) {
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "error reading response")
	}

	if response.StatusCode != http.StatusOK &&
		response.StatusCode != http.StatusCreated &&
		response.StatusCode != http.StatusNoContent {
		return responseData, c.handleErrorResponseFromSettingsAPI(responseData, response.StatusCode)
	}

	return responseData, nil
}

// GetEdgeConnect returns edge connect if it exists
func (c *client) GetEdgeConnect(edgeConnectID string) (GetResponse, error) {
	edgeConnectURL := c.getEdgeConnectURL(edgeConnectID)

	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, edgeConnectURL, nil)
	if err != nil {
		return GetResponse{}, err
	}

	resp, err := c.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(resp)

	if err != nil {
		return GetResponse{}, err
	}

	responseData, err := c.getServerResponseData(resp)
	if err != nil {
		return GetResponse{}, err
	}

	response := GetResponse{}

	err = json.Unmarshal(responseData, &response)
	if err != nil {
		return GetResponse{}, err
	}

	return response, nil
}

// UpdateEdgeConnect updates existing edge connect hostPatterns and oauthClientId
func (c *client) UpdateEdgeConnect(edgeConnectID string, request *Request) error {
	edgeConnectURL := c.getEdgeConnectURL(edgeConnectID)

	payloadBuf := new(bytes.Buffer)

	err := json.NewEncoder(payloadBuf).Encode(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPut, edgeConnectURL, payloadBuf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", contentTypeJSON)

	resp, err := c.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(resp)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse serverErrorResponse

		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			return err
		}

		return errors.Errorf("edgeconnect server error %d: %s: details %s", errorResponse.ErrorMessage.Code, errorResponse.ErrorMessage.Message, errorResponse.ErrorMessage.Details)
	}

	return nil
}

// DeleteEdgeConnect deletes edge connect using DELETE method for give edgeConnectId
func (c *client) DeleteEdgeConnect(edgeConnectID string) error {
	edgeConnectURL := c.getEdgeConnectURL(edgeConnectID)

	req, err := http.NewRequestWithContext(c.ctx, http.MethodDelete, edgeConnectURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(resp)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		var errorResponse serverErrorResponse

		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			return err
		}

		return errors.Errorf("edgeconnect server error %d: %s: details %s", errorResponse.ErrorMessage.Code, errorResponse.ErrorMessage.Message, errorResponse.ErrorMessage.Details)
	}

	return nil
}

// CreateEdgeConnect creates new edge connect
func (c *client) CreateEdgeConnect(request *Request) (CreateResponse, error) {
	edgeConnectsURL := c.getEdgeConnectsURL()

	payloadBuf := new(bytes.Buffer)

	err := json.NewEncoder(payloadBuf).Encode(request)
	if err != nil {
		return CreateResponse{}, err
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, edgeConnectsURL, payloadBuf)
	if err != nil {
		return CreateResponse{}, err
	}

	req.Header.Set("Content-Type", contentTypeJSON)

	resp, err := c.httpClient.Do(req)

	defer utils.CloseBodyAfterRequest(resp)

	if err != nil {
		return CreateResponse{}, err
	}

	responseData, err := c.getServerResponseData(resp)
	if err != nil {
		return CreateResponse{}, err
	}

	response := CreateResponse{}

	err = json.Unmarshal(responseData, &response)
	if err != nil {
		return CreateResponse{}, err
	}

	return response, nil
}

// GetEdgeConnects returns list of edge connects
func (c *client) GetEdgeConnects(name string) (ListResponse, error) {
	edgeConnectsURL := c.getEdgeConnectsURL()

	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, edgeConnectsURL, nil)
	if err != nil {
		return ListResponse{}, err
	}

	req.URL.RawQuery = url.Values{
		"add-fields": {"name,managedByDynatraceOperator"},
		"filter":     {fmt.Sprintf("name='%s'", name)},
	}.Encode()

	resp, err := c.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(resp)

	if err != nil {
		return ListResponse{}, err
	}

	responseData, err := c.getServerResponseData(resp)
	if err != nil {
		return ListResponse{}, err
	}

	response := ListResponse{}

	err = json.Unmarshal(responseData, &response)
	if err != nil {
		return ListResponse{}, err
	}

	return response, nil
}

// GetScopes returns all scopes that are used by the client
func (c *client) GetScopes() []string {
	return c.Scopes
}
