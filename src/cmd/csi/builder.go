package csi

import (
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/spf13/cobra"
)

const use = "csi-driver"

type commandBuilder struct {
	configProvider  config.Provider
	managerProvider cmdManager.Provider
	namespace       string
}

func newCsiCommandBuilder() commandBuilder {
	return commandBuilder{}
}

func (builder commandBuilder) setConfigProvider(provider config.Provider) commandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder commandBuilder) setManagerProvider(provider cmdManager.Provider) commandBuilder {
	builder.managerProvider = provider
	return builder
}

func (builder commandBuilder) setNamespace(namespace string) commandBuilder {
	builder.namespace = namespace
	return builder
}

func (builder commandBuilder) build() *cobra.Command {
	return &cobra.Command{
		Use:  use,
		RunE: builder.buildRun(),
	}
}

func (builder commandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		_, err = builder.managerProvider.CreateManager(builder.namespace, kubeConfig)
		if err != nil {
			return err
		}
		return nil
	}
}
