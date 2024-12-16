package registrar

import (
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/registrar"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

const use = "csi-registrar"

var endpoint, registrartionPath string

type CommandBuilder struct {
	// filesystem      afero.Fs
}

func NewCsiRegistrarCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

/*
	func (builder CommandBuilder) setFilesystem(filesystem afero.Fs) CommandBuilder {
		builder.filesystem = filesystem

		return builder
	}

	func (builder CommandBuilder) getFilesystem() afero.Fs {
		if builder.filesystem == nil {
			builder.filesystem = afero.NewOsFs()
		}

		return builder.filesystem
	}
*/

func (builder CommandBuilder) Build() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		RunE: builder.buildRun(),
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&endpoint, "csi-address", "/csi/csi.sock", "CSI endpoint")
	cmd.PersistentFlags().StringVar(&registrartionPath, "kubelet-registration-path", "/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com/csi.sock", "kubelet registration path")
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		version.LogVersion()
		logd.LogBaseLoggerSettings()

		signalHandler := ctrl.SetupSignalHandler()
		err := registrar.NewServer(dtcsi.DriverName, registrartionPath, []string{"v1.0.0"}).Start(signalHandler)

		return errors.WithStack(err)
	}
}
