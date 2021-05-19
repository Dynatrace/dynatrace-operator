package dtclient

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DynatracePaasToken = "paasToken"
	DynatraceApiToken  = "apiToken"
)

// Client is the interface for the Dynatrace REST API client.
type Client interface {
	// GetLatestAgentVersion gets the latest agent version for the given OS and installer type.
	// Returns the version as received from the server on success.
	//
	// Returns an error for the following conditions:
	//  - os or installerType is empty
	//  - IO error or unexpected response
	//  - error response from the server (e.g. authentication failure)
	//  - the agent version is not set or empty
	GetLatestAgentVersion(os, installerType string) (string, error)

	// GetLatestAgent returns a reader with the contents of the download. Must be closed by caller.
	GetLatestAgent(os, installerType, flavor, arch string) (io.ReadCloser, error)

	// GetCommunicationHosts returns, on success, the list of communication hosts used for available
	// communication endpoints that the Dynatrace OneAgent can use to connect to.
	//
	// Returns an error if there was also an error response from the server.
	GetConnectionInfo() (ConnectionInfo, error)

	// GetCommunicationHostForClient returns a CommunicationHost for the client's API URL. Or error, if failed to be parsed.
	GetCommunicationHostForClient() (CommunicationHost, error)

	// SendEvent posts events to dynatrace API
	SendEvent(eventData *EventData) error

	// GetEntityIDForIP returns the entity id for a given IP address.
	//
	// Returns an error in case the lookup failed.
	GetEntityIDForIP(ip string) (string, error)

	// GetTokenScopes returns the list of scopes assigned to a token if successful.
	GetTokenScopes(token string) (TokenScopes, error)

	// GetAgentTenantInfo returns TenantInfo that holds UUID, Tenant Token and Endpoints
	GetAgentTenantInfo() (*TenantInfo, error)

	// GetAGTenantInfo returns TenantInfo that holds UUID, Tenant Token and Endpoints
	GetAGTenantInfo() (*TenantInfo, error)
}

// Known OS values.
const (
	OsUnix = "unix"
	//Commented for linter, left for further reference
	//OsWindows = "windows"
	//OsAix     = "aix"
	//OsSolaris = "solaris"
)

// Known installer types.
const (
	InstallerTypeDefault = "default"
	//Commented for linter, left for further reference
	//InstallerTypeUnattended = "default-unattended"
	InstallerTypePaaS = "paas"
	//InstallerTypePaasSh     = "paas-sh"
)

// Known flavors.
const (
	FlavorDefault     = "default"
	FlavorMUSL        = "musl"
	FlavorMultidistro = "multidistro"
)

// Known architectures.
const (
	ArchX86 = "x86"
	ArchARM = "arm"
)

// Known token scopes
const (
	TokenScopeInstallerDownload = "InstallerDownload"
	TokenScopeDataExport        = "DataExport"
)

// NewClient creates a REST client for the given API base URL and authentication tokens.
// Returns an error if a token or the URL is empty.
//
// The API base URL is different for managed and SaaS environments:
//  - SaaS: https://{environment-id}.live.dynatrace.com/api
//  - Managed: https://{domain}/e/{environment-id}/api
//
// opts can be used to customize the created client, entries must not be nil.
func NewClient(url, apiToken, paasToken string, opts ...Option) (Client, error) {
	if len(url) == 0 {
		return nil, errors.New("url is empty")
	}
	if len(apiToken) == 0 && len(paasToken) == 0 {
		return nil, errors.New("tokens are empty")
	}

	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}

	dc := &dynatraceClient{
		url:       url,
		apiToken:  apiToken,
		paasToken: paasToken,
		logger:    log.Log.WithName("dynatrace.client"),

		hostCache: make(map[string]hostInfo),
		httpClient: &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		},
	}

	for _, opt := range opts {
		opt(dc)
	}
	return dc, nil
}

// Option can be passed to NewClient and customizes the created client instance.
type Option func(*dynatraceClient)

// SkipCertificateValidation creates an Option that specifies whether validation of the server's TLS
// certificate should be skipped. The default is false.
func SkipCertificateValidation(skip bool) Option {
	return func(c *dynatraceClient) {
		if skip {
			t := c.httpClient.Transport.(*http.Transport)
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = &tls.Config{}
			}
			t.TLSClientConfig.InsecureSkipVerify = true
		}
	}
}

func Proxy(proxyURL string) Option {
	return func(c *dynatraceClient) {
		p, err := url.Parse(proxyURL)
		if err != nil {
			c.logger.Info("Could not parse proxy URL!")
			return
		}
		t := c.httpClient.Transport.(*http.Transport)
		t.Proxy = http.ProxyURL(p)
	}
}

func Certs(certs []byte) Option {
	return func(c *dynatraceClient) {
		rootCAs := x509.NewCertPool()
		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			c.logger.Info("Failed to append custom certs!")
		}

		t := c.httpClient.Transport.(*http.Transport)
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		t.TLSClientConfig.RootCAs = rootCAs
	}
}

func NetworkZone(networkZone string) Option {
	return func(c *dynatraceClient) {
		c.networkZone = networkZone
	}
}

func DisableHostsRequests(disabledHostsRequests bool) Option {
	return func(c *dynatraceClient) {
		c.disableHostsRequests = disabledHostsRequests
	}
}
