package dynatrace

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/hostevent"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	operatorversion "github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"golang.org/x/net/http/httpproxy"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type Client struct {
	Settings   settings.APIClient
	ActiveGate activegate.APIClient
	HostEvent  hostevent.APIClient
	OneAgent   oneagent.APIClient
	Version    version.APIClient
	Token      token.APIClient
}

type OAuthClient struct {
	EdgeConnect edgeconnect.APIClient
}

type Config struct {
	APIToken    string
	PaasToken   string
	NetworkZone string
	HostGroup   string
	UserAgent   string

	BaseURL           *url.URL
	HTTPClient        *http.Client
	TLSConfig         *tls.Config
	Proxy             string
	NoProxy           string
	Timeout           time.Duration
	DisableKeepAlives bool
}

// Option is a functional option for configuring the client
type Option func(*Config) error

// NewClient creates a new Dynatrace API client
func NewClient(options ...Option) (*Client, error) {
	config, err := getConfig(options...)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client config")
	}

	if len(config.APIToken) == 0 && len(config.PaasToken) == 0 {
		return nil, errors.New("tokens are empty")
	}

	if dttoken.IsPlatform(config.APIToken) || config.PaasToken == "" {
		config.PaasToken = config.APIToken
	}

	if !strings.HasSuffix(strings.TrimSuffix(config.BaseURL.Path, "/"), "/api") {
		config.BaseURL.Path = strings.TrimSuffix(config.BaseURL.Path, "/") + "/api"
	}

	apiClient := core.NewClient(core.Config{
		BaseURL:    config.BaseURL,
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
		APIToken:   config.APIToken,
		PaasToken:  config.PaasToken,
	})

	return &Client{
		Settings:   settings.NewClient(apiClient),
		ActiveGate: activegate.NewClient(apiClient),
		HostEvent:  hostevent.NewClient(apiClient, config.NetworkZone),
		OneAgent:   oneagent.NewClient(apiClient, config.HostGroup, config.NetworkZone),
		Version:    version.NewClient(apiClient),
		Token:      token.NewClient(apiClient),
	}, nil
}

// NewOAuthClient creates a new Dynatrace API OAuth client
func NewOAuthClient(credentials clientcredentials.Config, options ...Option) (*OAuthClient, error) {
	config, err := getConfig(options...)
	if err != nil {
		return nil, errors.Wrap(err, "could not get oauth config")
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, config.HTTPClient)

	oAuthHTTPClient := credentials.Client(ctx)

	apiClient := core.NewClient(core.Config{
		BaseURL:    config.BaseURL,
		HTTPClient: oAuthHTTPClient,
		UserAgent:  config.UserAgent,
	})

	return &OAuthClient{
		EdgeConnect: edgeconnect.NewClient(apiClient),
	}, nil
}

// WithAPIToken sets the API token
func WithAPIToken(token string) Option {
	return func(c *Config) error {
		c.APIToken = token

		return nil
	}
}

// WithPaasToken sets the PaaS token
func WithPaasToken(token string) Option {
	return func(c *Config) error {
		c.PaasToken = token

		return nil
	}
}

// WithNetworkZone sets the network zone
func WithNetworkZone(networkZone string) Option {
	return func(c *Config) error {
		c.NetworkZone = networkZone

		return nil
	}
}

// WithHostGroup sets the host group
func WithHostGroup(hostGroup string) Option {
	return func(c *Config) error {
		c.HostGroup = hostGroup

		return nil
	}
}

// WithBaseURL parses the URL and sets it
func WithBaseURL(baseURL string) Option {
	return func(c *Config) error {
		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			return errors.Wrap(err, "invalid base URL")
		}

		if len(parsedURL.String()) == 0 {
			return errors.New("base URL is empty")
		}

		c.BaseURL = parsedURL

		return nil
	}
}

// WithUserAgentSuffix appends a suffix (comment) to the default user agent.
func WithUserAgentSuffix(suffix string) Option {
	return func(c *Config) error {
		if suffix != "" {
			c.UserAgent += " " + suffix
		}

		return nil
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Config) error {
		c.HTTPClient = httpClient

		return nil
	}
}

// WithProxy sets the proxy URL
func WithProxy(proxyURL, noProxy string) Option {
	return func(c *Config) error {
		c.Proxy = proxyURL
		c.NoProxy = noProxy

		return nil
	}
}

// WithTLSConfig sets custom TLS configuration
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(c *Config) error {
		c.TLSConfig = tlsConfig

		return nil
	}
}

// WithKeepAlive enables or disables HTTP keep-alives.
func WithKeepAlive(keepAlive bool) Option {
	return func(c *Config) error {
		c.DisableKeepAlives = !keepAlive

		return nil
	}
}

// WithSkipCertificateValidation skips TLS certificate validation when enabled.
func WithSkipCertificateValidation(skip bool) Option {
	return func(c *Config) error {
		if skip {
			if c.TLSConfig == nil {
				c.TLSConfig = &tls.Config{}
			}

			c.TLSConfig.InsecureSkipVerify = true
		}

		return nil
	}
}

// WithCerts appends custom root certificates to the system certificate pool.
func WithCerts(certs []byte) Option {
	return func(c *Config) error {
		if len(certs) == 0 {
			return nil
		}

		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			return errors.Wrap(err, "couldn't read system certificates")
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

func getConfig(options ...Option) (*Config, error) {
	config := Config{
		UserAgent: operatorversion.UserAgent(),
		Timeout:   30 * time.Second,
	}

	for _, opt := range options {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}

	t := http.DefaultTransport.(*http.Transport).Clone()

	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Transport: t,
		}
	} else {
		var ok bool

		t, ok = config.HTTPClient.Transport.(*http.Transport)
		if !ok {
			return nil, errors.New("unexpected transport type")
		}
	}

	t.TLSClientConfig = config.TLSConfig

	if config.Proxy != "" {
		proxyURL, err := url.Parse(config.Proxy)
		if err != nil {
			return nil, errors.Wrap(err, "invalid proxy URL")
		}

		proxyConfig := httpproxy.Config{
			HTTPProxy:  proxyURL.String(),
			HTTPSProxy: proxyURL.String(),
			NoProxy:    config.NoProxy,
		}
		t.Proxy = proxyWrapper(proxyConfig)
	}

	if config.DisableKeepAlives {
		t.DisableKeepAlives = true
	}

	if config.Timeout > 0 {
		config.HTTPClient.Timeout = config.Timeout
	}

	return &config, nil
}

func proxyWrapper(proxyConfig httpproxy.Config) func(req *http.Request) (*url.URL, error) {
	proxyFunc := proxyConfig.ProxyFunc()

	return func(req *http.Request) (*url.URL, error) { return proxyFunc(req.URL) }
}
