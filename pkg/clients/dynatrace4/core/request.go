package core

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

const APITokenHeader = "Api-Token "

type RequestBuilder interface {
	WithContext(ctx context.Context) RequestBuilder
	WithMethod(method string) RequestBuilder
	WithPath(path string) RequestBuilder
	WithQueryParam(key, value string) RequestBuilder
	WithQueryParams(params map[string]string) RequestBuilder
	WithJSONBody(body interface{}) RequestBuilder
	WithRawBody(body []byte) RequestBuilder
	WithTokenType(tokenType TokenType) RequestBuilder
	WithPaasToken() RequestBuilder
	Execute(target interface{}) error
	ExecuteRaw() ([]byte, error)
}

// requestBuilder provides a fluent interface for building and executing HTTP requests
type requestBuilder struct {
	config      CommonConfig
	method      string
	path        string
	queryParams map[string]string
	body        []byte
	tokenType   TokenType
	ctx         context.Context
}

// NewRequest creates a new RequestBuilder instance
func NewRequest(config CommonConfig) RequestBuilder {
	return &requestBuilder{
		config:      config,
		queryParams: make(map[string]string),
		tokenType:   TokenTypeAPI,
	}
}

// WithContext sets the context for the request
func (rb *requestBuilder) WithContext(ctx context.Context) RequestBuilder {
	rb.ctx = ctx
	return rb
}

// WithMethod sets the HTTP method for the request
func (rb *requestBuilder) WithMethod(method string) RequestBuilder {
	rb.method = method
	return rb
}

// WithPath sets the path for the request
func (rb *requestBuilder) WithPath(path string) RequestBuilder {
	rb.path = path
	return rb
}

// WithQueryParam adds a query parameter to the request
func (rb *requestBuilder) WithQueryParam(key, value string) RequestBuilder {
	rb.queryParams[key] = value
	return rb
}

// WithQueryParams adds multiple query parameters to the request
func (rb *requestBuilder) WithQueryParams(params map[string]string) RequestBuilder {
	for key, value := range params {
		rb.queryParams[key] = value
	}
	return rb
}

// WithJSONBody sets the request body as JSON
func (rb *requestBuilder) WithJSONBody(body interface{}) RequestBuilder {
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			// Store error for later handling during Execute
			rb.body = nil
		} else {
			rb.body = bodyBytes
		}
	}
	return rb
}

// WithRawBody sets the request body as raw bytes
func (rb *requestBuilder) WithRawBody(body []byte) RequestBuilder {
	rb.body = body
	return rb
}

// WithTokenType sets the token type to use for authentication
func (rb *requestBuilder) WithTokenType(tokenType TokenType) RequestBuilder {
	rb.tokenType = tokenType
	return rb
}

// WithPaasToken sets the token type to PaaS
func (rb *requestBuilder) WithPaasToken() RequestBuilder {
	rb.tokenType = TokenTypePaaS
	return rb
}

// Execute executes the request and unmarshals the response into the provided target
func (rb *requestBuilder) Execute(target interface{}) error {
	// Build URL
	reqURL, err := rb.config.BuildURL(rb.path, rb.queryParams)
	if err != nil {
		return errors.WithMessage(err, "failed to build URL")
	}

	// Create request
	var bodyReader io.Reader
	if rb.body != nil {
		bodyReader = bytes.NewReader(rb.body)
	}

	req, err := http.NewRequestWithContext(rb.ctx, rb.method, reqURL.String(), bodyReader)
	if err != nil {
		return errors.WithMessage(err, "failed to create HTTP request")
	}

	// Set headers
	rb.setHeaders(req)

	// Execute request
	resp, err := rb.config.HTTPClient.Do(req)
	if err != nil {
		return errors.WithMessage(err, "HTTP request failed")
	}
	defer utils.CloseBodyAfterRequest(resp)

	// Handle response
	return rb.handleResponse(resp, target)
}

// ExecuteRaw executes the request and returns the raw response data
func (rb *requestBuilder) ExecuteRaw() ([]byte, error) {
	// Build URL
	reqURL, err := rb.config.BuildURL(rb.path, rb.queryParams)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to build URL")
	}

	// Create request
	var bodyReader io.Reader
	if rb.body != nil {
		bodyReader = bytes.NewReader(rb.body)
	}

	req, err := http.NewRequestWithContext(rb.ctx, rb.method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create HTTP request")
	}

	// Set headers
	rb.setHeaders(req)

	// Execute request
	resp, err := rb.config.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.WithMessage(err, "HTTP request failed")
	}
	defer utils.CloseBodyAfterRequest(resp)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to read response body")
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, rb.handleErrorResponse(resp, body)
	}

	return body, nil
}

// setHeaders sets the common headers for the request
func (rb *requestBuilder) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", APITokenHeader+rb.config.GetToken(rb.tokenType))

	if rb.config.UserAgent != "" {
		req.Header.Set("User-Agent", rb.config.UserAgent)
	}

	if rb.method == http.MethodPost || rb.method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}
}

// handleResponse processes the HTTP response and unmarshals it into the target
func (rb *requestBuilder) handleResponse(resp *http.Response, target interface{}) error {
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.WithMessage(err, "failed to read response body")
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return rb.handleErrorResponse(resp, body)
	}

	// Unmarshal response if target is provided
	if target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return errors.WithMessage(err, "failed to unmarshal response body")
		}
	}

	return nil
}

// GET creates a GET request builder
func (config CommonConfig) GET(path string) RequestBuilder {
	return NewRequest(config).WithMethod(http.MethodGet).WithPath(path)
}

// POST creates a POST request builder
func (config CommonConfig) POST(path string) RequestBuilder {
	return NewRequest(config).WithMethod(http.MethodPost).WithPath(path)
}

// PUT creates a PUT request builder
func (config CommonConfig) PUT(path string) RequestBuilder {
	return NewRequest(config).WithMethod(http.MethodPut).WithPath(path)
}

// DELETE creates a DELETE request builder
func (config CommonConfig) DELETE(path string) RequestBuilder {
	return NewRequest(config).WithMethod(http.MethodDelete).WithPath(path)
}
