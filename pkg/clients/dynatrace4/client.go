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

type DtClient interface {
	Settings() settings.Client
	Token() token.Client
}

// dtClient is the main Dynatrace API dtClient that provides access to all API groups
type dtClient struct {

	// API clients for different groups
	// activeGateClient *activegate.dtClient
	// agentClient      *agent.dtClient
	// imagesClient     *images.dtClient
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

// Config holds the dtClient configuration
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

// Option is a functional option for configuring the dtClient
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

// WithHTTPClient sets a custom HTTP dtClient
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

// newClient creates a new Dynatrace API dtClient
func newClient(baseURL string, options ...Option) (DtClient, error) {
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

	// Create HTTP dtClient if not provided
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

	client := &dtClient{
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
func (c *dtClient) initAPIClients() {
	CoreClient := core.CoreClient{
		BaseURL:         c.baseURL,
		HTTPClient:      c.httpClient,
		UserAgent:       c.userAgent,
		APIToken:        c.apiToken,
		PaasToken:       c.paasToken,
		DataIngestToken: c.dataIngestToken,
		NetworkZone:     c.networkZone,
		HostGroup:       c.hostGroup,
	}

	// c.activeGateClient = activegate.NewClient(CoreClient)
	// c.agentClient = agent.NewClient(CoreClient)
	// c.imagesClient = images.NewClient(CoreClient)
	c.SettingsClient = settings.NewClient(CoreClient)
	c.TokenClient = token.NewClient(CoreClient)
}

// // ActiveGate returns the ActiveGate API dtClient
// func (c *dtClient) ActiveGate() *activegate.dtClient {
// 	return c.activeGateClient
// }

// // Agent returns the OneAgent API dtClient
// func (c *dtClient) Agent() *agent.dtClient {
// 	return c.agentClient
// }

// // Images returns the Images API dtClient
// func (c *dtClient) Images() *images.dtClient {
// 	return c.imagesClient
// }

// Settings returns the Settings API dtClient
func (c *dtClient) Settings() settings.Client {
	return c.SettingsClient
}

// Token returns the Token API dtClient
func (c *dtClient) Token() token.Client {
	return c.TokenClient
}

// BaseURL returns the base URL of the dtClient
func (c *dtClient) BaseURL() *url.URL {
	return c.baseURL
}

// HTTPClient returns the underlying HTTP dtClient
func (c *dtClient) HTTPClient() *http.Client {
	return c.httpClient
}
