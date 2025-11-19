package registrar

import (
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/csi/registrar"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

const use = "csi-node-driver-registrar"

var csiAddress, kubeletRegistrationPath, pluginRegistrationPath string

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		RunE: run,
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&csiAddress, "csi-address", "/csi/csi.sock", "CSI endpoint")
	cmd.PersistentFlags().StringVar(&pluginRegistrationPath, "plugin-registration-path", "/registration", "Kubernetes plugin registration path.")
	cmd.PersistentFlags().StringVar(&kubeletRegistrationPath, "kubelet-registration-path", "/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com/csi.sock", "Kubelet registration path.")
}

func run(*cobra.Command, []string) error {
	version.LogVersion()
	logd.LogBaseLoggerSettings()

	signalHandler := ctrl.SetupSignalHandler()
	err := registrar.NewServer(dtcsi.DriverName, csiAddress, kubeletRegistrationPath, pluginRegistrationPath, []string{"v1.0.0"}).Start(signalHandler)

	return errors.WithStack(err)
}
