package secret

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/config"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PrepareConfigFile(ctx context.Context, ec *edgeconnect.EdgeConnect, apiReader client.Reader, token string) ([]byte, error) {
	cfg := config.EdgeConnect{
		Name:            ec.Name,
		APIEndpointHost: ec.Spec.APIServer,
		OAuth: config.OAuth{
			Endpoint: ec.Spec.OAuth.Endpoint,
			Resource: ec.Spec.OAuth.Resource,
		},
		RestrictHostsTo: ec.Spec.HostRestrictions,
	}

	// For provisioned we need to read another secret, which later we mount to EdgeConnect pod
	if ec.IsProvisionerModeEnabled() {
		oAuth, err := ec.GetOAuthClientFromSecret(ctx, apiReader, ec.ClientSecretName())
		if err != nil {
			return []byte{}, err
		}

		cfg.OAuth.ClientID = oAuth.ClientID
		cfg.OAuth.ClientSecret = oAuth.ClientSecret
		cfg.OAuth.Resource = oAuth.Resource
	} else {
		// For regular, we use default secret
		oAuth, err := ec.GetOAuthClientFromSecret(ctx, apiReader, ec.Spec.OAuth.ClientSecret)
		if err != nil {
			return []byte{}, err
		}

		cfg.OAuth.ClientID = oAuth.ClientID
		cfg.OAuth.ClientSecret = oAuth.ClientSecret
	}

	if ec.Spec.CaCertsRef != "" {
		cfg.RootCertificatePaths = append(cfg.RootCertificatePaths, consts.EdgeConnectMountPath+"/"+consts.EdgeConnectCustomCertificateName)
	}

	// Always add certificates
	cfg.RootCertificatePaths = append(cfg.RootCertificatePaths, consts.EdgeConnectServiceAccountCAPath)

	if ec.IsK8SAutomationEnabled() {
		cfg.Secrets = append(cfg.Secrets, createKubernetesAPISecret(token))
	}

	if ec.Spec.Proxy != nil {
		cfg.Proxy = config.Proxy{
			Server:     ec.Spec.Proxy.Host,
			Port:       ec.Spec.Proxy.Port,
			Exceptions: ec.Spec.Proxy.NoProxy,
		}

		if ec.Spec.Proxy.AuthRef != "" {
			user, pass, err := ec.ProxyAuth(context.Background(), apiReader)
			if err != nil {
				return []byte{}, err
			}

			cfg.Proxy.Auth.User = user
			cfg.Proxy.Auth.Password = pass
		}
	}

	edgeConnectYaml, err := yaml.Marshal(cfg)
	log.Debug(safeEdgeConnectCfg(cfg))

	return edgeConnectYaml, err
}

// Replace client secret with stars for debug logs
func safeEdgeConnectCfg(cfg config.EdgeConnect) string {
	type safeOAuth struct {
		Endpoint string `yaml:"endpoint,omitempty"`
		ClientID string `yaml:"client_id,omitempty"`
		Resource string `yaml:"resource,omitempty"`
	}

	type safeProxy struct {
		User       string `yaml:"user,omitempty"`
		Server     string `yaml:"server,omitempty"`
		Exceptions string `yaml:"exceptions,omitempty"`
		Port       uint32 `yaml:"port,omitempty"`
	}

	type safeSecrets struct {
		Name            string   `yaml:"name,omitempty"`
		RestrictHostsTo []string `yaml:"restrict_hosts_to,omitempty"`
	}

	type safeConfig struct {
		Name                 string        `yaml:"name,omitempty"`
		APIEndpointHost      string        `yaml:"api_endpoint_host,omitempty"`
		OAuth                safeOAuth     `yaml:"oauth,omitempty"`
		RestrictHostsTo      []string      `yaml:"restrict_hosts_to,omitempty"`
		RootCertificatePaths []string      `yaml:"root_certificate_paths,omitempty"`
		Proxy                safeProxy     `yaml:"proxy,omitempty"`
		Secrets              []safeSecrets `yaml:"secrets,omitempty"`
	}

	safeCfg := safeConfig{
		Name:            cfg.Name,
		APIEndpointHost: cfg.APIEndpointHost,
		OAuth: safeOAuth{
			ClientID: cfg.OAuth.ClientID,
			Endpoint: cfg.OAuth.Endpoint,
			Resource: cfg.OAuth.Resource,
		},
		Proxy: safeProxy{
			User:       cfg.Proxy.Auth.User,
			Server:     cfg.Proxy.Server,
			Port:       cfg.Proxy.Port,
			Exceptions: cfg.Proxy.Exceptions,
		},
		RootCertificatePaths: cfg.RootCertificatePaths,
		RestrictHostsTo:      cfg.RestrictHostsTo,
	}

	for _, s := range cfg.Secrets {
		safeCfg.Secrets = append(safeCfg.Secrets, safeSecrets{Name: s.Name, RestrictHostsTo: s.RestrictHostsTo})
	}

	safe, _ := yaml.Marshal(safeCfg)

	return string(safe)
}

func createKubernetesAPISecret(token string) config.Secret {
	return config.Secret{
		Name:            "K8S_SERVICE_ACCOUNT_TOKEN",
		Token:           token,
		FromFile:        "/var/run/secrets/kubernetes.io/serviceaccount/token",
		RestrictHostsTo: []string{edgeconnect.KubernetesDefaultDNS},
	}
}
