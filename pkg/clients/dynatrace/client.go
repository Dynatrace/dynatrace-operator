package dynatrace

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
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
	ActiveGate activegate.Client
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
	TLSConfig         *tls.Config
	Proxy             string
	NoProxy           string
	DisableKeepAlives bool
	CacheEntryTTL     time.Duration
}

// Option is a functional option for configuring the client
type Option func(*Config) error

// NewClient creates a new Dynatrace API client
func NewClient(options ...Option) (*Client, error) {
	httpClient, config, err := getClientAndConfig(options...)
	if err != nil {
		return nil, errors.Wrap(err, "could not get client config")
	}

	addCacheMiddleware(httpClient, config)

	if len(config.APIToken) == 0 && len(config.PaasToken) == 0 {
		return nil, errors.New("tokens are empty")
	}

	if dttoken.IsPlatform(config.APIToken) || config.PaasToken == "" {
		config.PaasToken = config.APIToken
	}

	if strings.Contains(config.BaseURL.Hostname(), ".apps.") {
		mapThirdGenAPIURL(config.BaseURL)
	} else if path.Base(config.BaseURL.Path) != "api" {
		config.BaseURL = config.BaseURL.JoinPath("api")
	}

	apiClient := core.NewClient(core.Config{
		BaseURL:    config.BaseURL,
		HTTPClient: httpClient,
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
	httpClient, config, err := getClientAndConfig(options...)
	if err != nil {
		return nil, errors.Wrap(err, "could not get oauth config")
	}

	addCacheMiddleware(httpClient, config)

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)

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

func WithCacheTTL(ttl time.Duration) Option {
	return func(c *Config) error {
		c.CacheEntryTTL = ttl

		return nil
	}
}

func addCacheMiddleware(httpClient *http.Client, config *Config) {
	httpClient.Transport = middleware.NewCacheRoundTripper(httpClient.Transport, config.CacheEntryTTL)
}

func getClientAndConfig(options ...Option) (*http.Client, *Config, error) {
	config := Config{
		UserAgent: operatorversion.UserAgent(),
	}

	for _, opt := range options {
		if err := opt(&config); err != nil {
			return nil, nil, err
		}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()

	transport.TLSClientConfig = config.TLSConfig
	transport.DisableKeepAlives = config.DisableKeepAlives

	if config.Proxy != "" {
		proxyURL, err := url.Parse(config.Proxy)
		if err != nil {
			return nil, nil, errors.Wrap(err, "invalid proxy URL")
		}

		proxyConfig := httpproxy.Config{
			HTTPProxy:  proxyURL.String(),
			HTTPSProxy: proxyURL.String(),
			NoProxy:    config.NoProxy,
		}
		transport.Proxy = proxyWrapper(proxyConfig)
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, &config, nil
}

func proxyWrapper(proxyConfig httpproxy.Config) func(req *http.Request) (*url.URL, error) {
	proxyFunc := proxyConfig.ProxyFunc()

	return func(req *http.Request) (*url.URL, error) { return proxyFunc(req.URL) }
}

const thirdGenAPPSHostParts = 2

// mapThirdGenAPIURL remaps a 3rd gen URL (*.apps.*) to its 2nd gen equivalent.
func mapThirdGenAPIURL(u *url.URL) {
	hostname := u.Hostname()

	parts := strings.SplitN(hostname, ".apps.", thirdGenAPPSHostParts)
	if len(parts) != thirdGenAPPSHostParts {
		return
	}

	prefix, suffix := parts[0], parts[1]

	var newHostname string

	if !strings.Contains(prefix, ".") {
		newHostname = prefix + ".live." + suffix
	} else {
		newHostname = prefix + "." + suffix
	}

	if port := u.Port(); port != "" {
		u.Host = newHostname + ":" + port
	} else {
		u.Host = newHostname
	}

	u.Path = "/api"
}
