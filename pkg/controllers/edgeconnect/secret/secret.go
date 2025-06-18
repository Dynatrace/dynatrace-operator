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
		Name:            ec.ObjectMeta.Name,
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
	if cfg.OAuth.ClientSecret != "" {
		cfg.OAuth.ClientSecret = "****"
	}

	safe, _ := yaml.Marshal(cfg)

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
