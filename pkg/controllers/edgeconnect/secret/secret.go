package secret

import (
	"context"

	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/config"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PrepareConfigFile(ctx context.Context, instance *edgeconnectv1alpha1.EdgeConnect, apiReader client.Reader) ([]byte, error) {
	cfg := config.EdgeConnect{
		Name:            instance.ObjectMeta.Name,
		ApiEndpointHost: instance.Spec.ApiServer,
		OAuth: config.OAuth{
			Endpoint: instance.Spec.OAuth.Endpoint,
			Resource: instance.Spec.OAuth.Resource,
		},
		RestrictHostsTo: instance.Spec.HostRestrictions,
	}

	// For provisioned we need to read another secret, which later we mount to EdgeConnect pod
	if instance.Spec.OAuth.Provisioner {
		clientID, clientSecret, err := instance.GetOAuthClientFromSecret(ctx, apiReader, instance.ClientSecretName())
		if err != nil {
			return []byte{}, err
		}

		cfg.OAuth.ClientID = clientID
		cfg.OAuth.ClientSecret = clientSecret
	} else {
		// For regular, we use default secret
		clientID, clientSecret, err := instance.GetOAuthClientFromSecret(ctx, apiReader, instance.Spec.OAuth.ClientSecret)
		if err != nil {
			return []byte{}, err
		}

		cfg.OAuth.ClientID = clientID
		cfg.OAuth.ClientSecret = clientSecret
	}

	if instance.Spec.CaCertsRef != "" {
		cfg.RootCertificatePaths = append(cfg.RootCertificatePaths, consts.EdgeConnectMountPath+"/"+consts.EdgeConnectCustomCertificateName)
	}

	if instance.Spec.Proxy != nil {
		cfg.Proxy = config.Proxy{
			Server:     instance.Spec.Proxy.Host,
			Port:       instance.Spec.Proxy.Port,
			Exceptions: instance.Spec.Proxy.NoProxy,
		}

		if instance.Spec.Proxy.AuthRef != "" {
			user, pass, err := instance.ProxyAuth(context.Background(), apiReader)
			if err != nil {
				return []byte{}, err
			}

			cfg.Proxy.Auth.User = user
			cfg.Proxy.Auth.Password = pass
		}
	}

	edgeConnectYaml, err := yaml.Marshal(cfg)
	log.Debug(string(edgeConnectYaml))

	return edgeConnectYaml, err
}
