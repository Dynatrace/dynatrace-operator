package standalone

import (
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	use = "init"
)

func newStandaloneCommand() *cobra.Command {
	return &cobra.Command{
		Use:  use,
		RunE: startStandAloneInit,
	}
}

func startStandAloneInit(_ *cobra.Command, _ []string) error {
	standaloneRunner, err := standalone.NewRunner(afero.NewOsFs())
	if err != nil {
		return err
	}
	return standaloneRunner.Run()
}
