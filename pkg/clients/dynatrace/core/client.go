// Package core implements the base Dynatrace API client, shared utilities and types.
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const apiTokenHeader = "Api-Token "

var log = logd.Get().WithName("dtclient-core")

// APIClient defines the behavior required from a config provider and is mockable
type APIClient interface {
	GET(ctx context.Context, path string) APIRequest
	POST(ctx context.Context, path string) APIRequest
	PUT(ctx context.Context, path string) APIRequest
	DELETE(ctx context.Context, path string) APIRequest
}

// Cacheable must be implemented by types passed to Execute if they supposed to be cached.
// IsEmpty indicates whether the parsed response is considered empty.
// If IsEmpty returns true after a successful parse, the cache entry is removed.
type Cacheable interface {
	IsEmpty() bool
}

// APIRequest provides a fluent interface for building and executing HTTP requests
type APIRequest interface {
	// WithPath sets the path for the request. Path parts will be joined, ignoring leading or trailing slashes.
	WithPath(path ...string) APIRequest
	// WithQueryParams adds multiple query parameters to the request, overwriting existing keys if they exist
	WithQueryParams(params map[string]string) APIRequest
	// WithRawQueryParams adds multiple query parameters to the request
	WithRawQueryParams(params url.Values) APIRequest
	// WithJSONBody sets the request body as JSON
	WithJSONBody(body any) APIRequest
	// WithPaasToken sets the token type to PaaS
	WithPaasToken() APIRequest
	// WithoutToken explicitly disables authentication for the request
	WithoutToken() APIRequest
	// WithHeader sets a custom header for the request, overriding any default value
	WithHeader(key, value string) APIRequest
	// Execute executes the request and unmarshals the response into the provided model
	// If the provided model implements the Cacheable interface, then the client will cache the response.
	Execute(model any) error
	// ExecuteWriter executes the request, writes the response body to the provided writer,
	// and returns the response headers on success.
	ExecuteWriter(writer io.Writer) (http.Header, error)
}

type Config struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	UserAgent  string
	APIToken   string
	PaasToken  string
}

type Client struct {
	cfg Config
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
	}
}

type Request struct {
	client *Client

	ctx       context.Context
	query     url.Values
	headers   http.Header
	method    string
	path      string
	body      []byte
	tokenType TokenType
	err       error
}

// TokenType represents the type of authentication token to use
type TokenType int

const (
	TokenTypeAPI TokenType = iota
	TokenTypePaaS
	TokenTypeNone
)

func (c *Client) newRequest(ctx context.Context) *Request {
	headers := make(http.Header)

	query := make(url.Values)
	if c.cfg.BaseURL != nil {
		query = c.cfg.BaseURL.Query()
	}

	return &Request{
		headers: headers,
		client:  c,
		ctx:     ctx,
		query:   query,
	}
}

// GET creates a GET request builder
func (c *Client) GET(ctx context.Context, path string) APIRequest {
	return c.newRequest(ctx).withMethod(http.MethodGet).WithPath(path)
}

// POST creates a POST request builder
func (c *Client) POST(ctx context.Context, path string) APIRequest {
	return c.newRequest(ctx).withMethod(http.MethodPost).WithPath(path)
}

// PUT creates a PUT request builder
func (c *Client) PUT(ctx context.Context, path string) APIRequest {
	return c.newRequest(ctx).withMethod(http.MethodPut).WithPath(path)
}

// DELETE creates a DELETE request builder
func (c *Client) DELETE(ctx context.Context, path string) APIRequest {
	return c.newRequest(ctx).withMethod(http.MethodDelete).WithPath(path)
}

// WithPath sets the path for the request. Path parts will be joined, ignoring leading or trailing slashes
func (r *Request) WithPath(path ...string) APIRequest {
	r.path = (&url.URL{Path: r.path}).JoinPath(path...).Path

	return r
}

// WithQueryParams adds multiple query parameters to the request, overwriting existing keys if they exist
func (r *Request) WithQueryParams(params map[string]string) APIRequest {
	if r.query == nil {
		r.query = make(url.Values)
	}

	for key, value := range params {
		r.query.Set(key, value)
	}

	return r
}

// WithRawQueryParams adds multiple query parameters to the request
func (r *Request) WithRawQueryParams(params url.Values) APIRequest {
	if r.query == nil {
		r.query = make(url.Values)
	}

	for key, values := range params {
		for _, value := range values {
			r.query.Add(key, value)
		}
	}

	return r
}

// WithJSONBody sets the request body as JSON
func (r *Request) WithJSONBody(body any) APIRequest {
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			r.err = err
		} else {
			r.body = bodyBytes
		}
	}

	return r
}

// WithPaasToken sets the token type to PaaS
func (r *Request) WithPaasToken() APIRequest {
	r.tokenType = TokenTypePaaS

	return r
}

// WithoutToken explicitly disables authentication for the request
func (r *Request) WithoutToken() APIRequest {
	r.tokenType = TokenTypeNone

	return r
}

// WithHeader sets a custom header for the request, overriding existing value
func (r *Request) WithHeader(key, value string) APIRequest {
	r.headers.Set(key, value)

	return r
}

// Execute executes the request and unmarshals the response into the provided model
func (r *Request) Execute(model any) error {
	cacheableModel, isCacheable := model.(Cacheable)
	if isCacheable {
		r.headers.Set(middleware.CacheRequestHeader, "true")
	}

	body, cacheKey, err := r.doRequest()
	if err != nil {
		middleware.InvalidateCacheEntry(cacheKey)

		return err
	}

	if model != nil {
		if err := json.Unmarshal(body, model); err != nil {
			return fmt.Errorf("unmarshal response body: %w", err)
		}
	}

	if isCacheable && cacheableModel.IsEmpty() {
		middleware.InvalidateCacheEntry(cacheKey)
	}

	return nil
}

// ExecuteWriter executes the request, writes the response body to the provided writer,
// and returns the response headers on success.
func (r *Request) ExecuteWriter(writer io.Writer) (http.Header, error) {
	return r.doRequestStream(writer)
}

func (r *Request) getToken() string {
	switch r.tokenType {
	case TokenTypePaaS:
		return r.client.cfg.PaasToken
	case TokenTypeNone:
		return ""
	default:
		return r.client.cfg.APIToken
	}
}

func (r *Request) buildURL() (*url.URL, error) {
	if r.client.cfg.BaseURL == nil {
		return nil, errors.New("missing base URL")
	}

	u := r.client.cfg.BaseURL.JoinPath(r.path)

	if len(r.query) > 0 {
		u.RawQuery = r.query.Encode()
	}

	return u, nil
}

// WithMethod sets the HTTP method for the request
func (r *Request) withMethod(method string) APIRequest {
	r.method = method

	return r
}

func (r *Request) doRequestStream(writer io.Writer) (responseHeaders http.Header, err error) {
	if r.err != nil {
		return nil, r.err
	}

	reqURL, err := r.buildURL()
	if err != nil {
		return nil, fmt.Errorf("build URL: %w", err)
	}

	var bodyReader io.Reader
	if r.body != nil {
		bodyReader = bytes.NewReader(r.body)
	}

	req, err := http.NewRequestWithContext(r.ctx, r.method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}

	setHeaders(req, r.client.cfg.UserAgent, r.getToken(), r.headers)

	httpClient := r.client.cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	loggerArgs := createLoggerArgs(r.body)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			err = errors.Join(err, errClose)
		}
	}()

	if !IsSuccessResponse(resp) {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read error response body: %w", readErr)
		}

		log.Debug("API request", loggerArgs(resp, body)...)

		return nil, handleErrorResponse(resp, body)
	}

	log.Debug("API request", loggerArgs(resp, nil)...)

	if _, err = io.Copy(writer, resp.Body); err != nil {
		return nil, fmt.Errorf("stream response body: %w", err)
	}

	return resp.Header, nil
}

func (r *Request) doRequest() (body []byte, cacheKey string, err error) {
	if r.err != nil {
		return nil, "", r.err
	}

	reqURL, err := r.buildURL()
	if err != nil {
		return nil, "", fmt.Errorf("build URL: %w", err)
	}

	var bodyReader io.Reader
	if r.body != nil {
		bodyReader = bytes.NewReader(r.body)
	}

	req, err := http.NewRequestWithContext(r.ctx, r.method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, "", fmt.Errorf("create HTTP request: %w", err)
	}

	setHeaders(req, r.client.cfg.UserAgent, r.getToken(), r.headers)

	httpClient := r.client.cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	loggerArgs := createLoggerArgs(r.body)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("HTTP request: %w", err)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			err = errors.Join(err, errClose)
		}
	}()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response body: %w", err)
	}

	log.Debug("API request", loggerArgs(resp, body)...)

	if !IsSuccessResponse(resp) {
		err = handleErrorResponse(resp, body)
	}

	return body, resp.Header.Get(middleware.CacheKeyHeader), err
}

// setHeaders sets the common headers for the request
func setHeaders(req *http.Request, userAgent, token string, customHeaders http.Header) {
	req.Header.Set("Accept", "application/json")

	if token != "" {
		req.Header.Set("Authorization", apiTokenHeader+token)
	}

	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	if req.GetBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, values := range customHeaders {
		if len(values) == 0 {
			continue
		}

		req.Header.Set(key, values[0])
	}
}

// handleErrorResponse processes error responses from the API
func handleErrorResponse(resp *http.Response, body []byte) error {
	httpErr := &HTTPError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Message:    fmt.Sprintf("HTTP request (%s) failed %d", resp.Request.URL.Path, resp.StatusCode),
	}

	if isJSONList(body) {
		var errorArray []struct {
			ErrorMessage ServerError `json:"error"`
		}

		if err := json.Unmarshal(body, &errorArray); err == nil && len(errorArray) > 0 {
			httpErr.ServerErrors = make([]ServerError, len(errorArray))
			for i, errItem := range errorArray {
				httpErr.ServerErrors[i] = errItem.ErrorMessage
			}
		}
	} else {
		var singleError struct {
			Error ServerError `json:"error"`
		}

		if err := json.Unmarshal(body, &singleError); err == nil {
			httpErr.ServerErrors = []ServerError{singleError.Error}
		}
	}

	return httpErr
}

// IsSuccessResponse returns true when the HTTP response status code indicates
// a successful operation. DELETE requests accept 200-299; all other methods
// accept 200-201 (matching the legacy client behavior).
func IsSuccessResponse(resp *http.Response) bool {
	statusCodeThreshold := 201
	if resp.Request != nil && resp.Request.Method == http.MethodDelete {
		statusCodeThreshold = 299
	}

	return resp.StatusCode >= http.StatusOK && resp.StatusCode <= statusCodeThreshold
}

func isJSONList(body []byte) bool {
	// Dynatrace API can either return a list or a single JSON object depending on the request.
	// This is a simple way to check which is which by looking for JSON tokens and comparing their indices.
	sliceIdx := bytes.IndexByte(body, '[')
	if sliceIdx >= 0 {
		objIdx := bytes.IndexByte(body, '{')

		return sliceIdx < objIdx
	}

	return false
}
