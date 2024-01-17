package startup

import (
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
)

type dtclientBuilder struct {
	config     *SecretConfig
	trustedCAs string
	options    []dtclient.Option
}

func newDTClientBuilder(config *SecretConfig, trustedCAs string) *dtclientBuilder {
	return &dtclientBuilder{
		config:     config,
		trustedCAs: trustedCAs,
		options:    []dtclient.Option{},
	}
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
	builder.addHostGroup()
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

func (builder *dtclientBuilder) addHostGroup() {
	if builder.config.HostGroup != "" {
		builder.options = append(builder.options, dtclient.HostGroup(builder.config.HostGroup))
	}
}

func (builder *dtclientBuilder) addTrustedCerts() {
	if builder.trustedCAs != "" {
		log.Info("using TrustedCAs, check the secret for more details")

		builder.options = append(builder.options, dtclient.Certs([]byte(builder.trustedCAs)))
	}
}
