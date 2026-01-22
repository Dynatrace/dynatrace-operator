package dynatrace

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
)

type ClientV2 struct {
	Settings settings.APIClient
}

type ConfigV2 struct {
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

// OptionV2 is a functional option for configuring the dtClient
type OptionV2 func(*ConfigV2)

// WithAPIToken sets the API token
func WithAPIToken(token string) OptionV2 {
	return func(c *ConfigV2) {
		c.APIToken = token
	}
}

// WithPaasToken sets the PaaS token
func WithPaasToken(token string) OptionV2 {
	return func(c *ConfigV2) {
		c.PaasToken = token
	}
}

// WithDataIngestToken sets the data ingest token
func WithDataIngestToken(token string) OptionV2 {
	return func(c *ConfigV2) {
		c.DataIngestToken = token
	}
}

// WithHTTPClient sets a custom HTTP dtClient
func WithHTTPClient(httpClient *http.Client) OptionV2 {
	return func(c *ConfigV2) {
		c.HTTPClient = httpClient
	}
}

// WithTLSConfig sets custom TLS configuration
func WithTLSConfig(tlsConfig *tls.Config) OptionV2 {
	return func(c *ConfigV2) {
		c.TLSConfig = tlsConfig
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) OptionV2 {
	return func(c *ConfigV2) {
		c.Timeout = timeout
	}
}

// WithProxy sets the proxy URL
func WithProxy(proxyURL string) OptionV2 {
	return func(c *ConfigV2) {
		c.Proxy = proxyURL
	}
}

// WithNetworkZone sets the network zone
func WithNetworkZone(networkZone string) OptionV2 {
	return func(c *ConfigV2) {
		c.NetworkZone = networkZone
	}
}

// WithHostGroup sets the host group
func WithHostGroup(hostGroup string) OptionV2 {
	return func(c *ConfigV2) {
		c.HostGroup = hostGroup
	}
}

// newClientV2 creates a new Dynatrace API client
func newClientV2(baseURL string, options ...OptionV2) (*ClientV2, error) {
	config := ConfigV2{
		BaseURL:   baseURL,
		UserAgent: "dynatrace-operator/2.0",
		Timeout:   30 * time.Second,
	}

	for _, opt := range options {
		opt(&config)
	}

	parsedURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	if !strings.HasSuffix(strings.TrimSuffix(parsedURL.Path, "/"), "/api") {
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/") + "/api"
	}

	if config.HTTPClient == nil {
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

		config.HTTPClient = &http.Client{
			Transport: transport,
		}
	}

	if config.Timeout > 0 {
		config.HTTPClient.Timeout = config.Timeout
	}

	apiClient := core.NewClient(core.Config{
		BaseURL:         parsedURL,
		HTTPClient:      config.HTTPClient,
		UserAgent:       config.UserAgent,
		APIToken:        config.APIToken,
		PaasToken:       config.PaasToken,
		DataIngestToken: config.DataIngestToken,
	})

	return &ClientV2{
		Settings: settings.NewClient(apiClient),
	}, nil
}

func (dtc *dynatraceClient) AsV2() *ClientV2 {
	// Fields are already validated by the v1 client constructor
	v2, _ := newClientV2(
		dtc.url,
		WithAPIToken(dtc.apiToken),
		WithPaasToken(dtc.paasToken),
		WithDataIngestToken(""),
		WithNetworkZone(dtc.networkZone),
		WithHostGroup(dtc.hostGroup),
		WithHTTPClient(dtc.httpClient),
	)

	// Placeholders to prevent deadcode elimination
	// Will be used once the v1 HTTP client is no longer the default
	_ = WithProxy("")
	_ = WithTLSConfig(nil)
	_ = WithTimeout(0)

	return v2
}
