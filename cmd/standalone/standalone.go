package standalone

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/startup"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	use = "init"
)

func NewStandaloneCommand() *cobra.Command {
	return &cobra.Command{
		Use:          use,
		RunE:         startStandAloneInit,
		SilenceUsage: true,
	}
}

func startStandAloneInit(_ *cobra.Command, _ []string) error {
	unix.Umask(0000)

	signalHandler := ctrl.SetupSignalHandler()

	standaloneRunner, err := startup.NewRunner(afero.NewOsFs())
	if err != nil {
		return err
	}

	return standaloneRunner.Run(signalHandler)
}
