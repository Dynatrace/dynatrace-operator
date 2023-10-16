package edgeconnect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type client struct {
	ctx     context.Context
	baseURL string
	clientcredentials.Config
}

// Option can be passed to NewClient and customizes the created client instance.
type Option func(client *client)

// NewClient creates a REST client for the given API base URL
// opts can be used to customize the created client, entries must not be nil.
func NewClient(clientID, clientSecret string, options ...Option) (Client, error) {
	c := &client{
		Config: clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       DefaultOauthScopes,
			TokenURL:     DefaultTokenURL,
		},
	}

	for _, opt := range options {
		opt(c)
	}

	return c, nil
}

func NewClientWithProxy(clientID, clientSecret, proxy string) (Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		return proxyUrl, nil
	}
	httpClient := http.Client{Transport: transport}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)

	return NewClient(clientID, clientSecret, WithContext(ctx))
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

// WithContext can set context for client
// NB: via context you can override default http client to add Proxy or CA certificates
func WithContext(ctx context.Context) func(*client) {
	return func(c *client) {
		c.ctx = ctx
	}
}

// ServerError represents an error returned from the server (e.g. authentication failure).
type ServerError struct {
	Code    int          `json:"code,omitempty"`
	Message string       `json:"message,omitempty"`
	Details DetailsError `json:"details"`
}

// DetailsError represents details of errors
type DetailsError struct {
	ConstraintViolations ConstraintViolations `json:"constraintViolations"`
	MissingScopes        []string             `json:"missingScopes,omitempty"`
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

	return fmt.Sprintf("edgeconnect server error %d: %s", int64(e.Code), e.Message)
}

type serverErrorResponse struct {
	ErrorMessage ServerError `json:"error"`
}

func CloseBodyAfterRequest(response *http.Response) {
	if response != nil && response.Body != nil {
		err := response.Body.Close()
		if err != nil {
			return
		}
	}
}

func (c *client) handleErrorResponseFromAPI(response []byte, statusCode int) error {
	se := serverErrorResponse{}
	if err := json.Unmarshal(response, &se); err != nil {
		return errors.WithMessagef(err, "response error, can't unmarshal json response %d", statusCode)
	}

	return se.ErrorMessage
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

// GetEdgeConnect returns edge connect if it exists
func (c *client) GetEdgeConnect(edgeConnectId string) (GetResponse, error) {
	url := c.getEdgeConnectUrl(edgeConnectId)

	resp, err := c.Client(c.ctx).Get(url)

	if err != nil {
		return GetResponse{}, err
	}

	defer CloseBodyAfterRequest(resp)

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
func (c *client) UpdateEdgeConnect(edgeConnectId, name string, hostPatterns []string, oauthClientId string) error {
	url := c.getEdgeConnectUrl(edgeConnectId)

	body := &Request{
		Name:          name,
		HostPatterns:  hostPatterns,
		OauthClientId: oauthClientId,
	}
	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, url, payloadBuf)

	if err != nil {
		return err
	}

	resp, err := c.Client(c.ctx).Do(req)
	defer CloseBodyAfterRequest(resp)

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
func (c *client) DeleteEdgeConnect(edgeConnectId string) error {
	log.Info("started removing edge connect %s", edgeConnectId)
	url := c.getEdgeConnectUrl(edgeConnectId)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.Client(c.ctx).Do(req)
	defer CloseBodyAfterRequest(resp)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		var errorResponse serverErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			return err
		}
		return errors.Errorf("edgeconnect server error %d: %s", errorResponse.ErrorMessage.Code, errorResponse.ErrorMessage.Message)
	}
	log.Info("finished removing edge connect %s", edgeConnectId)
	return nil
}

// CreateEdgeConnect creates new edge connect
func (c *client) CreateEdgeConnect(name string, hostPatterns []string, oauthClientId string) (CreateResponse, error) {
	url := c.getEdgeConnectsUrl()

	body := &Request{
		Name:          name,
		HostPatterns:  hostPatterns,
		OauthClientId: oauthClientId,
	}
	payloadBuf := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuf).Encode(body)
	if err != nil {
		return CreateResponse{}, err
	}

	resp, err := c.Client(c.ctx).Post(url, http.MethodPost, payloadBuf)

	if err != nil {
		return CreateResponse{}, err
	}

	defer CloseBodyAfterRequest(resp)

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
