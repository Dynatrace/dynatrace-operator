package core

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"path"
)

// APIClient defines the behavior required from a config provider and is mockable.
type APIClient interface {
	GET(ctx context.Context, path string) RequestBuilder
	POST(ctx context.Context, path string) RequestBuilder
	PUT(ctx context.Context, path string) RequestBuilder
	DELETE(ctx context.Context, path string) RequestBuilder
}

// CommonConfig holds shared configuration for all API clients
type CommonConfig struct {
	BaseURL         *url.URL
	HTTPClient      *http.Client
	UserAgent       string
	APIToken        string
	PaasToken       string
	DataIngestToken string
	NetworkZone     string
	HostGroup       string
}

// TokenType represents the type of authentication token to use
type TokenType string

const (
	TokenTypeAPI        TokenType = "api"
	TokenTypePaaS       TokenType = "paas"
	TokenTypeDataIngest TokenType = "data-ingest"
)

// GetToken returns the appropriate token based on the token type
func (c CommonConfig) GetToken(tokenType TokenType) string {
	switch tokenType {
	case TokenTypePaaS:
		return c.PaasToken
	case TokenTypeDataIngest:
		return c.DataIngestToken
	default:
		return c.APIToken
	}
}

func (c CommonConfig) BuildURL(subPath string, queryParams map[string]string) (*url.URL, error) {
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

// GET creates a GET request builder
func (c CommonConfig) GET(ctx context.Context, path string) RequestBuilder {
	return newRequest(ctx, c).WithMethod(http.MethodGet).WithPath(path)
}

// POST creates a POST request builder
func (c CommonConfig) POST(ctx context.Context, path string) RequestBuilder {
	return newRequest(ctx, c).WithMethod(http.MethodPost).WithPath(path)
}

// PUT creates a PUT request builder
func (c CommonConfig) PUT(ctx context.Context, path string) RequestBuilder {
	return newRequest(ctx, c).WithMethod(http.MethodPut).WithPath(path)
}

// DELETE creates a DELETE request builder
func (c CommonConfig) DELETE(ctx context.Context, path string) RequestBuilder {
	return newRequest(ctx, c).WithMethod(http.MethodDelete).WithPath(path)
}

// CommunicationHost represents a communication endpoint
type CommunicationHost struct {
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}
