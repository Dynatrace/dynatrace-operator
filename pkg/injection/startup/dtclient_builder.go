package startup

import (
	"context"
	envclient "github.com/0sewa0/dynatrace-configuration-as-code-core/gen/environment"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"net/http"
)

type dtclientBuilder struct {
	config  *SecretConfig
	options []dtclient.Option
}

func newDTClientBuilder(config *SecretConfig) *dtclientBuilder {
	return &dtclientBuilder{
		config:  config,
		options: []dtclient.Option{},
	}
}

func createDtClient(ctx context.Context, apiUrl, apiToken string) *envclient.APIClient {
	tokenKey := "Api-Token"
	configuration := envclient.NewConfiguration()
	configuration.Servers = envclient.ServerConfigurations{{URL: apiUrl}}
	configuration.HTTPClient = http.DefaultClient
	apiClient := envclient.NewAPIClient(configuration)
	ctx = context.WithValue(ctx, envclient.ContextAPIKeys, map[string]envclient.APIKey{
		tokenKey: {
			Prefix: tokenKey,
			Key:    apiToken,
		},
	})
	return apiClient
}

func (builder *dtclientBuilder) createClient() (dtclient.Client, error) {
	log.Info("creating dtclient")
	builder.setOptions()
	client, err := dtclient.NewClient(
		builder.config.ApiUrl,
		builder.config.ApiToken,
		builder.config.PaasToken,
		builder.options...,
	)
	if err != nil {
		return nil, err
	}
	log.Info("dtclient created successfully")
	return client, nil
}

func (builder *dtclientBuilder) setOptions() {
	builder.addCertCheck()
	builder.addProxy()
	builder.addNetworkZone()
	builder.addTrustedCerts()
}

func (builder *dtclientBuilder) addCertCheck() {
	if builder.config.SkipCertCheck {
		log.Info("skip cert check is enabled")
		builder.options = append(builder.options, dtclient.SkipCertificateValidation(true))
	}
}

func (builder *dtclientBuilder) addProxy() {
	if builder.config.Proxy != "" {
		log.Info("using the following proxy", "proxy", builder.config.Proxy)
		builder.options = append(builder.options, dtclient.Proxy(builder.config.Proxy, builder.config.NoProxy))
	}
}

func (builder *dtclientBuilder) addNetworkZone() {
	if builder.config.NetworkZone != "" {
		builder.options = append(builder.options, dtclient.NetworkZone(builder.config.NetworkZone))
	}
}

func (builder *dtclientBuilder) addTrustedCerts() {
	if builder.config.TrustedCAs != "" {
		log.Info("using TrustedCAs, check the secret for more details")
		builder.options = append(builder.options, dtclient.Certs([]byte(builder.config.TrustedCAs)))
	}
}
