package startup

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
)

type dtclientBuilder struct {
	config  *SecretConfig
	options []dynatrace.Option
}

func newDTClientBuilder(config *SecretConfig) *dtclientBuilder {
	return &dtclientBuilder{
		config:  config,
		options: []dynatrace.Option{},
	}
}

func (builder *dtclientBuilder) createClient() (dynatrace.Client, error) {
	log.Info("creating dtclient")
	builder.setOptions()
	client, err := dynatrace.NewClient(
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
		builder.options = append(builder.options, dynatrace.SkipCertificateValidation(true))
	}
}

func (builder *dtclientBuilder) addProxy() {
	if builder.config.Proxy != "" {
		log.Info("using the following proxy", "proxy", builder.config.Proxy)
		builder.options = append(builder.options, dynatrace.Proxy(builder.config.Proxy, builder.config.NoProxy))
	}
}

func (builder *dtclientBuilder) addNetworkZone() {
	if builder.config.NetworkZone != "" {
		builder.options = append(builder.options, dynatrace.NetworkZone(builder.config.NetworkZone))
	}
}

func (builder *dtclientBuilder) addTrustedCerts() {
	if builder.config.TrustedCAs != "" {
		log.Info("using TrustedCAs, check the secret for more details")
		builder.options = append(builder.options, dynatrace.Certs([]byte(builder.config.TrustedCAs)))
	}
}
