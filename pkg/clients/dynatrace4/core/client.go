package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	pkgerrors "github.com/pkg/errors"
)

const APITokenHeader = "Api-Token "

// APIClient defines the behavior required from a config provider and is mockable.
type APIClient interface {
	GET(ctx context.Context, path string) RequestBuilder
	POST(ctx context.Context, path string) RequestBuilder
	PUT(ctx context.Context, path string) RequestBuilder
	DELETE(ctx context.Context, path string) RequestBuilder
}

// RequestBuilder provides a fluent interface for building and executing HTTP requests
type RequestBuilder interface {
	WithPath(path string) RequestBuilder
	WithQueryParams(params map[string]string) RequestBuilder
	WithJSONBody(body interface{}) RequestBuilder
	WithRawBody(body []byte) RequestBuilder
	WithTokenType(tokenType TokenType) RequestBuilder
	WithPaasToken() RequestBuilder
	Execute(target interface{}) error
	ExecuteRaw() ([]byte, error)
}

// CoreClient holds both configuration and request state, serving as both config provider and request builder
type CoreClient struct {
	// Configuration fields
	BaseURL         *url.URL
	HTTPClient      *http.Client
	UserAgent       string
	APIToken        string
	PaasToken       string
	DataIngestToken string
	NetworkZone     string
	HostGroup       string

	// Request state fields
	ctx         context.Context
	queryParams map[string]string
	method      string
	path        string
	tokenType   TokenType
	body        []byte
}

// TokenType represents the type of authentication token to use
type TokenType string

const (
	TokenTypeAPI        TokenType = "api"
	TokenTypePaaS       TokenType = "paas"
	TokenTypeDataIngest TokenType = "data-ingest"
)

// GetToken returns the appropriate token based on the token type
func (c CoreClient) GetToken(tokenType TokenType) string {
	switch tokenType {
	case TokenTypePaaS:
		return c.PaasToken
	case TokenTypeDataIngest:
		return c.DataIngestToken
	default:
		return c.APIToken
	}
}

func (c CoreClient) BuildURL(subPath string, queryParams map[string]string) (*url.URL, error) {
	if c.BaseURL == nil {
		return nil, errors.New("base URL is not set")
	}

	u := *c.BaseURL
	// Join the base path and the provided subPath, preserving /api
	u.Path = path.Join(u.Path, subPath)

	if len(queryParams) > 0 {
		q := u.Query()
		for key, value := range queryParams {
			q.Set(key, value)
		}

		u.RawQuery = q.Encode()
	}

	return &u, nil
}

// newRequest creates a new Client instance for building a request
func (c CoreClient) newRequest(ctx context.Context) *CoreClient {
	return &CoreClient{
		BaseURL:         c.BaseURL,
		HTTPClient:      c.HTTPClient,
		UserAgent:       c.UserAgent,
		APIToken:        c.APIToken,
		PaasToken:       c.PaasToken,
		DataIngestToken: c.DataIngestToken,
		NetworkZone:     c.NetworkZone,
		HostGroup:       c.HostGroup,
		ctx:             ctx,
		queryParams:     make(map[string]string),
		tokenType:       TokenTypeAPI,
	}
}

// GET creates a GET request builder
func (c CoreClient) GET(ctx context.Context, path string) RequestBuilder {
	return c.newRequest(ctx).withMethod(http.MethodGet).WithPath(path)
}

// POST creates a POST request builder
func (c CoreClient) POST(ctx context.Context, path string) RequestBuilder {
	return c.newRequest(ctx).withMethod(http.MethodPost).WithPath(path)
}

// PUT creates a PUT request builder
func (c CoreClient) PUT(ctx context.Context, path string) RequestBuilder {
	return c.newRequest(ctx).withMethod(http.MethodPut).WithPath(path)
}

// DELETE creates a DELETE request builder
func (c CoreClient) DELETE(ctx context.Context, path string) RequestBuilder {
	return c.newRequest(ctx).withMethod(http.MethodDelete).WithPath(path)
}

// WithMethod sets the HTTP method for the request
func (c *CoreClient) withMethod(method string) RequestBuilder {
	c.method = method

	return c
}

// WithPath sets the path for the request
func (c *CoreClient) WithPath(path string) RequestBuilder {
	c.path = path

	return c
}

// WithQueryParams adds multiple query parameters to the request
func (c *CoreClient) WithQueryParams(params map[string]string) RequestBuilder {
	if c.queryParams == nil {
		c.queryParams = make(map[string]string)
	}

	for key, value := range params {
		c.queryParams[key] = value
	}

	return c
}

// WithJSONBody sets the request body as JSON
func (c *CoreClient) WithJSONBody(body interface{}) RequestBuilder {
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			// Store error for later handling during Execute
			c.body = nil
		} else {
			c.body = bodyBytes
		}
	}

	return c
}

// WithRawBody sets the request body as raw bytes
func (c *CoreClient) WithRawBody(body []byte) RequestBuilder {
	c.body = body

	return c
}

// WithTokenType sets the token type to use for authentication
func (c *CoreClient) WithTokenType(tokenType TokenType) RequestBuilder {
	c.tokenType = tokenType

	return c
}

// WithPaasToken sets the token type to PaaS
func (c *CoreClient) WithPaasToken() RequestBuilder {
	c.tokenType = TokenTypePaaS

	return c
}

// Execute executes the request and unmarshals the response into the provided target
func (c *CoreClient) Execute(target interface{}) error {
	// Build URL
	reqURL, err := c.BuildURL(c.path, c.queryParams)
	if err != nil {
		return pkgerrors.WithMessage(err, "failed to build URL")
	}

	// Create request
	var bodyReader io.Reader
	if c.body != nil {
		bodyReader = bytes.NewReader(c.body)
	}

	req, err := http.NewRequestWithContext(c.ctx, c.method, reqURL.String(), bodyReader)
	if err != nil {
		return pkgerrors.WithMessage(err, "failed to create HTTP request")
	}

	// Set headers
	c.setHeaders(req)

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return pkgerrors.WithMessage(err, "HTTP request failed")
	}
	defer utils.CloseBodyAfterRequest(resp)

	// Handle response
	return c.handleResponse(resp, target)
}

// ExecuteRaw executes the request and returns the raw response data
func (c *CoreClient) ExecuteRaw() ([]byte, error) {
	// Build URL
	reqURL, err := c.BuildURL(c.path, c.queryParams)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "failed to build URL")
	}

	// Create request
	var bodyReader io.Reader
	if c.body != nil {
		bodyReader = bytes.NewReader(c.body)
	}

	req, err := http.NewRequestWithContext(c.ctx, c.method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "failed to create HTTP request")
	}

	// Set headers
	c.setHeaders(req)

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "HTTP request failed")
	}
	defer utils.CloseBodyAfterRequest(resp)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "failed to read response body")
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, c.handleErrorResponse(resp, body)
	}

	return body, nil
}

// setHeaders sets the common headers for the request
func (c *CoreClient) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", APITokenHeader+c.GetToken(c.tokenType))

	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	if c.method == http.MethodPost || c.method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}
}

// handleResponse processes the HTTP response and unmarshals it into the target
func (c *CoreClient) handleResponse(resp *http.Response, target interface{}) error {
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return pkgerrors.WithMessage(err, "failed to read response body")
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.handleErrorResponse(resp, body)
	}

	// Unmarshal response if target is provided
	if target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return pkgerrors.WithMessage(err, "failed to unmarshal response body")
		}
	}

	return nil
}

// CommunicationHost represents a communication endpoint
type CommunicationHost struct {
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}
