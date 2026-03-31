package dynatrace

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/hostevent"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/version"
	operatorversion "github.com/Dynatrace/dynatrace-operator/pkg/version"
	"golang.org/x/net/http/httpproxy"
)

type ClientV2 struct {
	Settings   settings.APIClient
	ActiveGate activegate.APIClient
	HostEvent  hostevent.APIClient
	OneAgent   oneagent.APIClient
	Version    version.APIClient
	Token      token.APIClient
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
	NoProxy         string
	NetworkZone     string
	HostGroup       string
	Timeout         time.Duration
}

// OptionV2 is a functional option for configuring the dtClient
type OptionV2 func(*ConfigV2) error

// WithAPIToken sets the API token
func WithAPIToken(token string) OptionV2 {
	return func(c *ConfigV2) error {
		c.APIToken = token

		return nil
	}
}

// WithPaasToken sets the PaaS token
func WithPaasToken(token string) OptionV2 {
	return func(c *ConfigV2) error {
		c.PaasToken = token

		return nil
	}
}

// WithDataIngestToken sets the data ingest token
func WithDataIngestToken(token string) OptionV2 {
	return func(c *ConfigV2) error {
		c.DataIngestToken = token

		return nil
	}
}

// WithHTTPClient sets a custom HTTP dtClient
func WithHTTPClient(httpClient *http.Client) OptionV2 {
	return func(c *ConfigV2) error {
		c.HTTPClient = httpClient

		return nil
	}
}

// WithTLSConfig sets custom TLS configuration
func WithTLSConfig(tlsConfig *tls.Config) OptionV2 {
	return func(c *ConfigV2) error {
		c.TLSConfig = tlsConfig

		return nil
	}
}

func WithSkipCertificateValidation(skip bool) OptionV2 {
	return func(c *ConfigV2) error {
		if skip {
			if c.TLSConfig == nil {
				c.TLSConfig = &tls.Config{}
			}

			c.TLSConfig.InsecureSkipVerify = true
		}

		return nil
	}
}

func WithCerts(certs []byte) OptionV2 {
	return func(c *ConfigV2) error {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			return fmt.Errorf("couldn't read system certificates: %w", err)
		}

		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			return errors.New("failed to append custom certs")
		}

		if c.TLSConfig == nil {
			c.TLSConfig = &tls.Config{}
		}

		c.TLSConfig.RootCAs = rootCAs

		return nil
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) OptionV2 {
	return func(c *ConfigV2) error {
		c.Timeout = timeout

		return nil
	}
}

// WithProxy sets the proxy URL
func WithProxy(proxyURL, noProxy string) OptionV2 {
	return func(c *ConfigV2) error {
		c.Proxy = proxyURL
		c.NoProxy = noProxy

		return nil
	}
}

// WithNetworkZone sets the network zone
func WithNetworkZone(networkZone string) OptionV2 {
	return func(c *ConfigV2) error {
		c.NetworkZone = networkZone

		return nil
	}
}

// WithHostGroup sets the host group
func WithHostGroup(hostGroup string) OptionV2 {
	return func(c *ConfigV2) error {
		c.HostGroup = hostGroup

		return nil
	}
}

// WithUserAgentSuffix appends a suffix (comment) to the default user agent.
func WithUserAgentSuffix(suffix string) OptionV2 {
	return func(c *ConfigV2) error {
		if suffix != "" {
			c.UserAgent += " " + suffix
		}

		return nil
	}
}

// NewClientV2 creates a new Dynatrace API client
func NewClientV2(baseURL string, options ...OptionV2) (*ClientV2, error) {
	config := ConfigV2{
		BaseURL:   baseURL,
		UserAgent: operatorversion.UserAgent(),
		Timeout:   30 * time.Second,
	}

	for _, opt := range options {
		if err := opt(&config); err != nil {
			return nil, err
		}
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

			proxyConfig := httpproxy.Config{
				HTTPProxy:  proxyURL.String(),
				HTTPSProxy: proxyURL.String(),
				NoProxy:    config.NoProxy,
			}
			transport.Proxy = func(req *http.Request) (*url.URL, error) {
				return proxyConfig.ProxyFunc()(req.URL)
			}
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
		Settings:   settings.NewClient(apiClient),
		ActiveGate: activegate.NewClient(apiClient),
		HostEvent:  hostevent.NewClient(apiClient, config.NetworkZone),
		OneAgent:   oneagent.NewClient(apiClient, config.HostGroup, config.NetworkZone),
		Version:    version.NewClient(apiClient),
		Token:      token.NewClient(apiClient),
	}, nil
}

func (dtc *dynatraceClient) AsV2() *ClientV2 {
	// Fields are already validated by the v1 client constructor
	v2, _ := NewClientV2(
		dtc.url,
		WithUserAgentSuffix(dtc.userAgentSuffix),
		WithAPIToken(dtc.apiToken),
		WithPaasToken(dtc.paasToken),
		WithDataIngestToken(""),
		WithNetworkZone(dtc.networkZone),
		WithHostGroup(dtc.hostGroup),
		WithHTTPClient(dtc.httpClient),
	)

	// Placeholders to prevent deadcode elimination
	// Will be used once the v1 HTTP client is no longer the default
	_ = WithTLSConfig(nil)
	_ = WithTimeout(0)

	return v2
}
