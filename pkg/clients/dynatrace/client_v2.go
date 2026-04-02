package dynatrace

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
	operatorversion "github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"golang.org/x/net/http/httpproxy"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
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
	HTTPClient  *http.Client
	BaseURL     string
	APIToken    string
	PaasToken   string
	NetworkZone string
	HostGroup   string
	UserAgent   string
	baseOptions []BaseOption
}

// OptionV2 is a functional option for configuring the v2 client
type OptionV2 func(*ConfigV2) error

// NewClientV2 creates a new Dynatrace V2 API client
func NewClientV2(baseURL string, options ...OptionV2) (*ClientV2, error) {
	config := ConfigV2{
		BaseURL:   baseURL,
		UserAgent: operatorversion.UserAgent(),
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

	config.HTTPClient, err = newBaseClient(config.baseOptions...)
	if err != nil {
		return nil, errors.Wrap(err, "could not create http client")
	}

	apiClient := core.NewClient(core.Config{
		BaseURL:    parsedURL,
		HTTPClient: config.HTTPClient,
		UserAgent:  config.UserAgent,
		APIToken:   config.APIToken,
		PaasToken:  config.PaasToken,
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

// WithV2UserAgentSuffix appends a suffix (comment) to the default user agent.
func WithV2UserAgentSuffix(suffix string) OptionV2 {
	return func(c *ConfigV2) error {
		if suffix != "" {
			c.UserAgent += " " + suffix
		}

		return nil
	}
}

// WithV2BaseOptions adds the http base options
func WithV2BaseOptions(options ...BaseOption) OptionV2 {
	return func(c *ConfigV2) error {
		c.baseOptions = append(c.baseOptions, options...)

		return nil
	}
}

type OAuthClient struct {
	EdgeConnect edgeconnect.APIClient
}

type OAuthConfig struct {
	HTTPClient *http.Client
	ctx        context.Context
	clientcredentials.Config
	BaseURL     string
	UserAgent   string
	baseOptions []BaseOption
}

// OAuthOption is a functional option for configuring the OAuth client
type OAuthOption func(client *OAuthConfig) error

// NewOAuthClient creates a new Dynatrace API OAuth client
func NewOAuthClient(baseURL string, options ...OAuthOption) (*OAuthClient, error) {
	config := OAuthConfig{
		BaseURL:   baseURL,
		UserAgent: operatorversion.UserAgent(),
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

	config.HTTPClient, err = newBaseClient(config.baseOptions...)
	if err != nil {
		return nil, errors.Wrap(err, "could not create http client")
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, config.HTTPClient)

	oAuthHTTPClient := config.Client(ctx)

	apiClient := core.NewClient(core.Config{
		BaseURL:    parsedURL,
		HTTPClient: oAuthHTTPClient,
		UserAgent:  config.UserAgent,
	})

	return &OAuthClient{
		EdgeConnect: edgeconnect.NewClient(apiClient),
	}, nil
}

// WithClientID sets the OAuth client ID.
func WithClientID(id string) OAuthOption {
	return func(c *OAuthConfig) error {
		c.ClientID = id

		return nil
	}
}

// WithClientSecret sets the OAuth client secret.
func WithClientSecret(secret string) OAuthOption {
	return func(c *OAuthConfig) error {
		c.ClientSecret = secret

		return nil
	}
}

// WithTokenURL sets the OAuth token URL.
func WithTokenURL(url string) OAuthOption {
	return func(c *OAuthConfig) error {
		c.TokenURL = url

		return nil
	}
}

// WithOAuthScopes sets the OAuth scopes.
func WithOAuthScopes(scopes []string) OAuthOption {
	return func(c *OAuthConfig) error {
		c.Scopes = scopes

		return nil
	}
}

// WithContext sets the context used for OAuth client configuration.
func WithContext(ctx context.Context) OAuthOption {
	return func(c *OAuthConfig) error {
		c.ctx = ctx

		return nil
	}
}

// WithOAuthUserAgentSuffix appends a suffix (comment) to the default user agent.
func WithOAuthUserAgentSuffix(suffix string) OAuthOption {
	return func(c *OAuthConfig) error {
		if suffix != "" {
			c.UserAgent += " " + suffix
		}

		return nil
	}
}

// WithOAuthBaseOptions adds HTTP base options for the OAuth client.
func WithOAuthBaseOptions(options ...BaseOption) OAuthOption {
	return func(c *OAuthConfig) error {
		c.baseOptions = append(c.baseOptions, options...)

		return nil
	}
}

type BaseConfig struct {
	HTTPClient        *http.Client
	TLSConfig         *tls.Config
	Proxy             string
	NoProxy           string
	Timeout           time.Duration
	DisableKeepAlives bool
}

// BaseOption is a functional option for configuring the base HTTP client.
type BaseOption func(*BaseConfig) error

// newBaseClient creates an HTTP client configured with the provided base options.
func newBaseClient(options ...BaseOption) (*http.Client, error) {
	config := BaseConfig{
		Timeout: 30 * time.Second,
	}

	for _, opt := range options {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}

	t := &http.Transport{}

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
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
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

	return config.HTTPClient, nil
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) BaseOption {
	return func(c *BaseConfig) error {
		c.HTTPClient = httpClient

		return nil
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) BaseOption {
	return func(c *BaseConfig) error {
		c.Timeout = timeout

		return nil
	}
}

// WithProxy sets the proxy URL
func WithProxy(proxyURL, noProxy string) BaseOption {
	return func(c *BaseConfig) error {
		c.Proxy = proxyURL
		c.NoProxy = noProxy

		return nil
	}
}

// WithTLSConfig sets custom TLS configuration
func WithTLSConfig(tlsConfig *tls.Config) BaseOption {
	return func(c *BaseConfig) error {
		c.TLSConfig = tlsConfig

		return nil
	}
}

// WithKeepAlive enables or disables HTTP keep-alives.
func WithKeepAlive(keepAlive bool) BaseOption {
	return func(c *BaseConfig) error {
		c.DisableKeepAlives = !keepAlive

		return nil
	}
}

// WithSkipCertificateValidation skips TLS certificate validation when enabled.
func WithSkipCertificateValidation(skip bool) BaseOption {
	return func(c *BaseConfig) error {
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
func WithCerts(certs []byte) BaseOption {
	return func(c *BaseConfig) error {
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

func (dtc *dynatraceClient) AsV2() *ClientV2 {
	// Fields are already validated by the v1 client constructor
	v2, _ := NewClientV2(
		dtc.url,
		WithV2UserAgentSuffix(dtc.userAgentSuffix),
		WithAPIToken(dtc.apiToken),
		WithPaasToken(dtc.paasToken),
		WithNetworkZone(dtc.networkZone),
		WithHostGroup(dtc.hostGroup),
		WithV2BaseOptions(WithHTTPClient(dtc.httpClient)),
	)

	// Placeholders to prevent deadcode elimination
	// Will be used once the v1 HTTP client is no longer the default
	_ = WithTLSConfig(nil)
	_ = WithTimeout(0)

	return v2
}

func proxyWrapper(proxyConfig httpproxy.Config) func(req *http.Request) (*url.URL, error) {
	proxyFunc := proxyConfig.ProxyFunc()

	return func(req *http.Request) (*url.URL, error) { return proxyFunc(req.URL) }
}
