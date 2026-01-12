// Package core implements the base Dynatrace API client, shared utilities and types.
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
)

const apiTokenHeader = "Api-Token "

// APIClient defines the behavior required from a config provider and is mockable
type APIClient interface {
	GET(ctx context.Context, path string) APIRequest
	POST(ctx context.Context, path string) APIRequest
	PUT(ctx context.Context, path string) APIRequest
	DELETE(ctx context.Context, path string) APIRequest
}

// APIRequest provides a fluent interface for building and executing HTTP requests
type APIRequest interface {
	WithPath(path string) APIRequest
	WithQueryParams(params map[string]string) APIRequest
	WithJSONBody(body any) APIRequest
	WithRawBody(body []byte) APIRequest
	WithTokenType(tokenType TokenType) APIRequest
	WithPaasToken() APIRequest
	Execute(target any) error
	ExecuteRaw() ([]byte, error)
}

type Config struct {
	BaseURL         *url.URL
	HTTPClient      *http.Client
	UserAgent       string
	APIToken        string
	PaasToken       string
	DataIngestToken string
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

	ctx         context.Context
	queryParams map[string]string
	method      string
	path        string
	body        []byte
	tokenType   TokenType
	err         error
}

// TokenType represents the type of authentication token to use
type TokenType int

const (
	TokenTypeAPI TokenType = iota
	TokenTypePaaS
	TokenTypeDataIngest
)

func (c *Client) newRequest(ctx context.Context) *Request {
	return &Request{
		client:      c,
		ctx:         ctx,
		queryParams: make(map[string]string),
		tokenType:   TokenTypeAPI,
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

// WithPath sets the path for the request
func (r *Request) WithPath(path string) APIRequest {
	r.path = path

	return r
}

// WithQueryParams adds multiple query parameters to the request
func (r *Request) WithQueryParams(params map[string]string) APIRequest {
	if r.queryParams == nil {
		r.queryParams = make(map[string]string)
	}

	maps.Copy(r.queryParams, params)

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

// WithRawBody sets the request body as raw bytes
func (r *Request) WithRawBody(body []byte) APIRequest {
	r.body = body

	return r
}

// WithTokenType sets the token type to use for authentication
func (r *Request) WithTokenType(tokenType TokenType) APIRequest {
	r.tokenType = tokenType

	return r
}

// WithPaasToken sets the token type to PaaS
func (r *Request) WithPaasToken() APIRequest {
	r.tokenType = TokenTypePaaS

	return r
}

// Execute executes the request and unmarshals the response into the provided target
func (r *Request) Execute(target any) error {
	resp, err := r.doRequest()
	if err != nil {
		return err
	}
	defer utils.CloseBodyAfterRequest(resp)

	return handleResponse(resp, target)
}

// ExecuteRaw executes the request and returns the raw response data
func (r *Request) ExecuteRaw() ([]byte, error) {
	resp, err := r.doRequest()
	if err != nil {
		return nil, err
	}
	defer utils.CloseBodyAfterRequest(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, handleErrorResponse(resp, body)
	}

	return body, nil
}

func (r *Request) getToken() string {
	switch r.tokenType {
	case TokenTypePaaS:
		return r.client.cfg.PaasToken
	case TokenTypeDataIngest:
		return r.client.cfg.DataIngestToken
	default:
		return r.client.cfg.APIToken
	}
}

func (r *Request) buildURL() (*url.URL, error) {
	if r.client.cfg.BaseURL == nil {
		return nil, errors.New("missing base URL")
	}

	u := r.client.cfg.BaseURL.JoinPath(r.path)

	if len(r.queryParams) > 0 {
		q := u.Query()
		for key, value := range r.queryParams {
			q.Set(key, value)
		}

		u.RawQuery = q.Encode()
	}

	return u, nil
}

// WithMethod sets the HTTP method for the request
func (r *Request) withMethod(method string) APIRequest {
	r.method = method

	return r
}

func (r *Request) doRequest() (*http.Response, error) {
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

	setHeaders(req, r.client.cfg.UserAgent, r.getToken())

	httpClient := r.client.cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}

	return resp, nil
}

// setHeaders sets the common headers for the request
func setHeaders(req *http.Request, userAgent, token string) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", apiTokenHeader+token)

	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	if req.GetBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
}

// handleResponse processes the HTTP response and unmarshals it into the target
func handleResponse(resp *http.Response, target any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return handleErrorResponse(resp, body)
	}

	if target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("unmarshal response body: %w", err)
		}
	}

	return nil
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
