package dynatrace

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/pkg/errors"
	"golang.org/x/net/http/httpproxy"
)

const (
	PaasToken       = "paasToken"
	APIToken        = "apiToken"
	DataIngestToken = "dataIngestToken"
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
	GetLatestAgentVersion(ctx context.Context, os, installerType string) (string, error)

	// GetLatestAgent returns a reader with the contents of the download. Must be closed by caller.
	GetLatestAgent(ctx context.Context, os, installerType, flavor, arch string, technologies []string, skipMetadata bool, writer io.Writer) error

	// GetAgent downloads a specific agent version and writes it to the given io.Writer
	GetAgent(ctx context.Context, os, installerType, flavor, arch, version string, technologies []string, skipMetadata bool, writer io.Writer) error

	// GetAgentViaInstallerUrl downloads the agent from the user specified URL and writes it to the given io.Writer
	GetAgentViaInstallerURL(ctx context.Context, url string, writer io.Writer) error

	// GetAgentVersions on success returns an array of versions that can be used with GetAgent to
	// download a specific agent version
	GetAgentVersions(ctx context.Context, os, installerType, flavor string) ([]string, error)

	GetOneAgentConnectionInfo(ctx context.Context) (OneAgentConnectionInfo, error)

	GetProcessModuleConfig(ctx context.Context, prevRevision uint) (*ProcessModuleConfig, error)

	// GetCommunicationHostForClient returns a CommunicationHost for the client's API URL. Or error, if failed to be parsed.
	GetCommunicationHostForClient() (CommunicationHost, error)

	// SendEvent posts events to dynatrace API
	SendEvent(ctx context.Context, eventData *EventData) error

	// GetHostEntityIDForIP returns the host entity id for a given IP address.
	// Returns an error in case the lookup failed.
	GetHostEntityIDForIP(ctx context.Context, ip string) (string, error)

	// GetTokenScopes returns the list of scopes assigned to a token if successful.
	GetTokenScopes(ctx context.Context, token string) (TokenScopes, error)

	// GetActiveGateConnectionInfo returns AgentTenantInfo for ActiveGate that holds UUID, Tenant Token and Endpoints
	GetActiveGateConnectionInfo(ctx context.Context) (ActiveGateConnectionInfo, error)

	// CreateOrUpdateKubernetesSetting returns the object id of the created k8s settings if successful, or an api error otherwise
	CreateOrUpdateKubernetesSetting(ctx context.Context, name, kubeSystemUUID, scope string) (string, error)

	// CreateLogMonitoringSetting returns the object id of the created logmonitoring settings if successful, or an api error otherwise
	CreateLogMonitoringSetting(ctx context.Context, scope, clusterName string, ingestRuleMatchers []logmonitoring.IngestRuleMatchers) (string, error)

	// CreateOrUpdateKubernetesAppSetting returns the object id of the created k8s app settings if successful, or an api error otherwise
	CreateOrUpdateKubernetesAppSetting(ctx context.Context, scope string) (string, error)

	// GetK8sClusterME returns the Kubernetes Cluster Monitored Entity for the give kubernetes cluster.
	// Uses the `settings.read` scope to list the `builtin:cloud.kubernetes` settings.
	// - Only 1 such setting exists per tenant per kubernetes cluster
	// - The `scope` for the setting is the ID (example: KUBERNETES_CLUSTER-A1234567BCD8EFGH) of the Kubernetes Cluster Monitored Entity
	// - The `label` of the setting is the Name (example: my-dynakube) of the Kubernetes Cluster Monitored Entity
	// In case 0 settings are found, so no Kubernetes Cluster Monitored Entity exist, we return an empty object, without an error.
	GetK8sClusterME(ctx context.Context, kubeSystemUUID string) (K8sClusterME, error)

	// GetSettingsForMonitoredEntity returns the settings response with the number of settings objects,
	// or an api error otherwise
	GetSettingsForMonitoredEntity(ctx context.Context, monitoredEntity K8sClusterME, schemaID string) (GetSettingsResponse, error)

	// GetSettingsForLogModule returns the settings response with the number of settings objects,
	// or an api error otherwise
	GetSettingsForLogModule(ctx context.Context, monitoredEntity string) (GetLogMonSettingsResponse, error)

	GetRulesSettings(ctx context.Context, kubeSystemUUID string, entityID string) (GetRulesSettingsResponse, error)

	GetActiveGateAuthToken(ctx context.Context, dynakubeName string) (*ActiveGateAuthTokenInfo, error)

	GetLatestOneAgentImage(ctx context.Context) (*LatestImageInfo, error)

	GetLatestCodeModulesImage(ctx context.Context) (*LatestImageInfo, error)

	GetLatestActiveGateImage(ctx context.Context) (*LatestImageInfo, error)

	// GetLatestActiveGateVersion gets the latest gateway version for the given OS and arch.
	// Returns the version as received from the server on success.
	GetLatestActiveGateVersion(ctx context.Context, os string) (string, error)
}

const (
	OsUnix = "unix"
	// Commented for linter, left for further reference
	// OsWindows = "windows"
	// OsAix     = "aix"
	// OsSolaris = "solaris"
)

// Relevant installer types.
const (
	InstallerTypeDefault = "default"
	InstallerTypePaaS    = "paas"
)

// Relevant token scopes
const (
	TokenScopeInstallerDownload        = "InstallerDownload"
	TokenScopeDataExport               = "DataExport"
	TokenScopeMetricsIngest            = "metrics.ingest"
	TokenScopeOpenTelemetryTraceIngest = "openTelemetryTrace.ingest"
	TokenScopeLogsIngest               = "logs.ingest"
	TokenScopeSettingsRead             = "settings.read"
	TokenScopeSettingsWrite            = "settings.write"
	TokenScopeActiveGateTokenCreate    = "activeGateTokenManagement.create"
)

const (
	ConditionTypeAPITokenSettingsRead  = "ApiTokenSettingsRead"
	ConditionTypeAPITokenSettingsWrite = "ApiTokenSettingsWrite"
)

var (
	OptionalScopes = map[string]string{
		TokenScopeSettingsRead:  ConditionTypeAPITokenSettingsRead,
		TokenScopeSettingsWrite: ConditionTypeAPITokenSettingsWrite,
	}
)

type NewFunc func(url, apiToken, paasToken string, opts ...Option) (Client, error)

var _ NewFunc = NewClient

// NewClient creates a REST client for the given API base URL and authentication tokens.
// Returns an error if a token or the URL is empty.
//
// The API base URL is different for managed and SaaS environments:
//   - SaaS: https://{environment-id}.live.dynatrace.com/api
//   - Managed: https://{domain}/e/{environment-id}/api
//
// opts can be used to customize the created client, entries must not be nil.
func NewClient(url, apiToken, paasToken string, opts ...Option) (Client, error) {
	if len(url) == 0 {
		return nil, errors.New("url is empty")
	}

	if len(apiToken) == 0 && len(paasToken) == 0 {
		return nil, errors.New("tokens are empty")
	}

	url = strings.TrimSuffix(url, "/")

	dc := &dynatraceClient{
		url:       url,
		apiToken:  apiToken,
		paasToken: paasToken,

		hostCache: make(map[string]hostInfo),
		httpClient: &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
			Timeout:   15 * time.Minute,
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
				t.TLSClientConfig = &tls.Config{} //nolint:gosec // fix is expected to be delivered soon
			}

			t.TLSClientConfig.InsecureSkipVerify = true
		}
	}
}

func Proxy(proxyURL string, noProxy string) Option {
	return func(dtclient *dynatraceClient) {
		if proxyURL == "" {
			return
		}

		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			log.Info("could not parse proxy URL!")

			return
		}

		transport := dtclient.httpClient.Transport.(*http.Transport)
		proxyConfig := httpproxy.Config{
			HTTPProxy:  parsedURL.String(),
			HTTPSProxy: parsedURL.String(),
			NoProxy:    noProxy,
		}
		transport.Proxy = proxyWrapper(proxyConfig)
	}
}

func proxyWrapper(proxyConfig httpproxy.Config) func(req *http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		return proxyConfig.ProxyFunc()(req.URL)
	}
}

func Certs(certs []byte) Option {
	return func(c *dynatraceClient) {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			log.Info("couldn't read system certificates!")

			return
		}

		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			log.Info("failed to append custom certs!")
		}

		t := c.httpClient.Transport.(*http.Transport)
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{} //nolint:gosec // fix is expected to be delivered soon
		}

		t.TLSClientConfig.RootCAs = rootCAs
	}
}

func NetworkZone(networkZone string) Option {
	return func(c *dynatraceClient) {
		c.networkZone = networkZone
	}
}

func HostGroup(hostGroup string) Option {
	return func(c *dynatraceClient) {
		c.hostGroup = hostGroup
	}
}
