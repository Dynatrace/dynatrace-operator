package csi

import (
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/spf13/cobra"
)

const use = "csi-driver"

type commandBuilder struct {
	configProvider config.Provider
}

func newCsiCommandBuilder() commandBuilder {
	return commandBuilder{}
}

func (builder commandBuilder) setConfigProvider(provider config.Provider) commandBuilder {
	builder.configProvider = provider
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
		_, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		return nil
	}
}
