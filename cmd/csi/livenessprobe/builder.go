package livenessprobe

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/livenessprobe"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

const use = "csi-livenessprobe"

var probeTimeout, endpoint, healthPort string

type CommandBuilder struct {
	// filesystem      afero.Fs
}

func NewCsiLivenessprobeCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) Build() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		RunE: builder.buildRun(),
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&probeTimeout, "probe-timeout", "4s", "probe timeout")
	cmd.PersistentFlags().StringVar(&endpoint, "csi-address", "/csi/csi.sock", "CSI endpoint")
	cmd.PersistentFlags().StringVar(&healthPort, "health-port", "9808", "health port")
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		version.LogVersion()
		logd.LogBaseLoggerSettings()

		signalHandler := ctrl.SetupSignalHandler()
		err := livenessprobe.NewServer(endpoint, healthPort, probeTimeout).Start(signalHandler)

		return errors.WithStack(err)
	}
}
