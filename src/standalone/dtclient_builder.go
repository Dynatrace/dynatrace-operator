package standalone

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
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

func (builder *dtclientBuilder) createClient() (dtclient.Client, error) {
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
		builder.options = append(builder.options, dtclient.SkipCertificateValidation(true))
	}
}

func (builder *dtclientBuilder) addProxy() {
	if builder.config.Proxy != "" {
		builder.options = append(builder.options, dtclient.Proxy(builder.config.Proxy))
	}
}

func (builder *dtclientBuilder) addNetworkZone() {
	if builder.config.NetworkZone != "" {
		builder.options = append(builder.options, dtclient.NetworkZone(builder.config.NetworkZone))
	}
}

func (builder *dtclientBuilder) addTrustedCerts() {
	if builder.config.TrustedCAs != "" {
		builder.options = append(builder.options, dtclient.Certs([]byte(builder.config.TrustedCAs)))
	}
}
