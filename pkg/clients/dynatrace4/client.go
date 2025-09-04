package dynatrace4

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4/token"
)

const (
	PaasToken       = "paasToken"
	APIToken        = "apiToken"
	DataIngestToken = "dataIngestToken"
)

// Client is the main Dynatrace API client that provides access to all API groups
type Client struct {

	// API clients for different groups
	// activeGateClient *activegate.Client
	// agentClient      *agent.Client
	// imagesClient     *images.Client
	SettingsClient settings.Client
	TokenClient    token.Client
	baseURL        *url.URL
	httpClient     *http.Client
	userAgent      string

	// API tokens
	apiToken        string
	paasToken       string
	dataIngestToken string

	// Additional configuration
	networkZone string
	hostGroup   string
}

// Config holds the client configuration
type Config struct {
	HTTPClient      *http.Client
	TLSConfig       *tls.Config
	BaseURL         string
	APIToken        string
	PaasToken       string
	DataIngestToken string
	UserAgent       string
	Proxy           string
	NetworkZone     string
	HostGroup       string
	Timeout         time.Duration
}

// Option is a functional option for configuring the client
type Option func(*Config)

// WithAPIToken sets the API token
func WithAPIToken(token string) Option {
	return func(c *Config) {
		c.APIToken = token
	}
}

// WithPaasToken sets the PaaS token
func WithPaasToken(token string) Option {
	return func(c *Config) {
		c.PaasToken = token
	}
}

// WithDataIngestToken sets the data ingest token
func WithDataIngestToken(token string) Option {
	return func(c *Config) {
		c.DataIngestToken = token
	}
}

// WithUserAgent sets a custom user agent
func WithUserAgent(userAgent string) Option {
	return func(c *Config) {
		c.UserAgent = userAgent
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = httpClient
	}
}

// WithTLSConfig sets custom TLS configuration
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(c *Config) {
		c.TLSConfig = tlsConfig
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithProxy sets the proxy URL
func WithProxy(proxyURL string) Option {
	return func(c *Config) {
		c.Proxy = proxyURL
	}
}

// WithNetworkZone sets the network zone
func WithNetworkZone(networkZone string) Option {
	return func(c *Config) {
		c.NetworkZone = networkZone
	}
}

// WithHostGroup sets the host group
func WithHostGroup(hostGroup string) Option {
	return func(c *Config) {
		c.HostGroup = hostGroup
	}
}

// NewClient creates a new Dynatrace API client
func NewClient(baseURL string, options ...Option) (*Client, error) {
	config := &Config{
		BaseURL:   baseURL,
		UserAgent: "dynatrace-operator/2.0",
		Timeout:   30 * time.Second,
	}

	// Apply options
	for _, opt := range options {
		opt(config)
	}

	// Parse and validate base URL
	parsedURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Ensure the URL ends with /api if not already present
	if !strings.HasSuffix(strings.TrimSuffix(parsedURL.Path, "/"), "/api") {
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/") + "/api"
	}

	// Create HTTP client if not provided
	var httpClient *http.Client
	if config.HTTPClient != nil {
		httpClient = config.HTTPClient
	} else {
		transport := &http.Transport{
			TLSClientConfig: config.TLSConfig,
		}

		// Configure proxy if provided
		if config.Proxy != "" {
			proxyURL, err := url.Parse(config.Proxy)
			if err != nil {
				return nil, fmt.Errorf("invalid proxy URL: %w", err)
			}

			transport.Proxy = http.ProxyURL(proxyURL)
		}

		httpClient = &http.Client{
			Transport: transport,
			Timeout:   config.Timeout,
		}
	}

	client := &Client{
		baseURL:         parsedURL,
		httpClient:      httpClient,
		userAgent:       config.UserAgent,
		apiToken:        config.APIToken,
		paasToken:       config.PaasToken,
		dataIngestToken: config.DataIngestToken,
		networkZone:     config.NetworkZone,
		hostGroup:       config.HostGroup,
	}

	// Initialize API clients
	client.initAPIClients()

	return client, nil
}

// initAPIClients initializes all the API group clients
func (c *Client) initAPIClients() {
	commonConfig := core.CommonConfig{
		BaseURL:         c.baseURL,
		HTTPClient:      c.httpClient,
		UserAgent:       c.userAgent,
		APIToken:        c.apiToken,
		PaasToken:       c.paasToken,
		DataIngestToken: c.dataIngestToken,
		NetworkZone:     c.networkZone,
		HostGroup:       c.hostGroup,
	}

	// c.activeGateClient = activegate.NewClient(commonConfig)
	// c.agentClient = agent.NewClient(commonConfig)
	// c.imagesClient = images.NewClient(commonConfig)
	c.SettingsClient = settings.NewClient(commonConfig)
	c.TokenClient = token.NewClient(commonConfig)
}

// // ActiveGate returns the ActiveGate API client
// func (c *Client) ActiveGate() *activegate.Client {
// 	return c.activeGateClient
// }

// // Agent returns the OneAgent API client
// func (c *Client) Agent() *agent.Client {
// 	return c.agentClient
// }

// // Images returns the Images API client
// func (c *Client) Images() *images.Client {
// 	return c.imagesClient
// }

// Settings returns the Settings API client
func (c *Client) Settings() settings.Client {
	return c.SettingsClient
}

// Token returns the Token API client
func (c *Client) Token() token.Client {
	return c.TokenClient
}

// BaseURL returns the base URL of the client
func (c *Client) BaseURL() *url.URL {
	return c.baseURL
}

// HTTPClient returns the underlying HTTP client
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// // DoRequest performs an HTTP request with proper authentication
// func (c *Client) DoRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
// 	return c.DoRequestWithToken(ctx, method, path, body, c.apiToken)
// }

// // DoRequestWithToken performs an HTTP request with a specific token
// func (c *Client) DoRequestWithToken(ctx context.Context, method, path string, body interface{}, token string) (*http.Response, error) {
// 	fullURL := c.baseURL.ResolveReference(&url.URL{Path: path})

// 	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create request: %w", err)
// 	}

// 	// Set headers
// 	req.Header.Set("User-Agent", c.userAgent)
// 	req.Header.Set("Authorization", "Api-Token "+token)
// 	req.Header.Set("Content-Type", "application/json")

// 	return c.httpClient.Do(req)
// }
