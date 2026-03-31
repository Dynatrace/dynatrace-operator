package edgeconnect

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// APIClient is the interface for the Dynatrace EdgeConnect REST API client.
type APIClient interface {
	// GetEdgeConnect return details of single edge connect
	GetEdgeConnect(ctx context.Context, edgeConnectID string) (GetResponse, error)

	// CreateEdgeConnect creates edge connect
	CreateEdgeConnect(ctx context.Context, request *Request) (CreateResponse, error)

	// UpdateEdgeConnect updates edge connect
	UpdateEdgeConnect(ctx context.Context, edgeConnectID string, request *Request) error

	// DeleteEdgeConnect deletes edge connect
	DeleteEdgeConnect(ctx context.Context, edgeConnectID string) error

	// GetEdgeConnects returns list of edge connects
	GetEdgeConnects(ctx context.Context, name string) (ListResponse, error)

	// GetConnectionSettings returns all connection setting objects
	GetConnectionSettings(ctx context.Context) ([]EnvironmentSetting, error)

	// CreateConnectionSetting creates a connection setting object
	CreateConnectionSetting(ctx context.Context, es EnvironmentSetting) error

	// UpdateConnectionSetting updates a connection setting object
	UpdateConnectionSetting(ctx context.Context, es EnvironmentSetting) error

	// DeleteConnectionSetting deletes a connection setting object
	DeleteConnectionSetting(ctx context.Context, objectID string) error
}

type client struct {
	apiClient core.APIClient
}

type builder struct {
	ctx               context.Context
	cfg               core.Config
	oauthCfg          clientcredentials.Config
	customCA          []byte
	disableKeepAlives bool
	baseURL           string
}

// Option can be passed to NewClient and customizes the created client instance.
type Option func(b *builder)

func NewClient(ops ...Option) (APIClient, error) {
	b := &builder{
		ctx: context.Background(),
	}

	for _, op := range ops {
		op(b)
	}

	apiClient, err := b.buildCoreClient()
	if err != nil {
		return nil, err
	}

	return &client{
		apiClient: apiClient,
	}, nil
}

func NewClientFromAPIClient(apiClient core.APIClient) APIClient {
	return &client{
		apiClient: apiClient,
	}
}

func WithClientID(id string) func(*builder) {
	return func(b *builder) {
		b.oauthCfg.ClientID = id
	}
}

func WithClientSecret(secret string) func(*builder) {
	return func(b *builder) {
		b.oauthCfg.ClientSecret = secret
	}
}

func WithBaseURL(url string) func(*builder) {
	return func(b *builder) {
		b.baseURL = url
	}
}

func WithTokenURL(url string) func(*builder) {
	return func(b *builder) {
		b.oauthCfg.TokenURL = url
	}
}

func WithOAuthScopes(scopes []string) func(*builder) {
	return func(b *builder) {
		b.oauthCfg.Scopes = scopes
	}
}

func WithCustomCA(caData []byte) func(*builder) {
	return func(b *builder) {
		b.customCA = caData
	}
}

func WithKeepAlive(keepAlive bool) func(*builder) {
	return func(b *builder) {
		b.disableKeepAlives = !keepAlive
	}
}

func WithContext(ctx context.Context) func(*builder) {
	return func(b *builder) {
		b.ctx = ctx
	}
}

func (b *builder) buildCoreClient() (core.APIClient, error) {
	parsedURL, err := url.Parse(b.baseURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	b.cfg.BaseURL = parsedURL

	httpClient := b.oauthCfg.Client(b.ctx)

	if httpClient == nil {
		return nil, errors.New("can't create http client for edge connect")
	}

	ot, ok := httpClient.Transport.(*oauth2.Transport)
	if !ok {
		return nil, errors.New("unexpected transport type")
	}

	if ot.Base == nil {
		ot.Base = &http.Transport{}
	}

	if b.disableKeepAlives {
		if t, ok := ot.Base.(*http.Transport); ok {
			t.DisableKeepAlives = true
		}
	}

	if b.customCA != nil {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			return nil, errors.Wrap(err, "read system certificates")
		}

		if ok := rootCAs.AppendCertsFromPEM(b.customCA); !ok {
			return nil, errors.New("append custom certs")
		}

		t := ot.Base.(*http.Transport)
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}

		t.TLSClientConfig.RootCAs = rootCAs
	}

	b.cfg.HTTPClient = httpClient

	return core.NewClient(b.cfg), nil
}
