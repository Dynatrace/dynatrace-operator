package livenessprobe

import (
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/livenessprobe"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	use = "livenessprobe"

	defaultProbeTimeout = 9 * time.Second
)

var (
	probeTimeout           time.Duration
	csiAddress, healthPort string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		RunE: run,
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().DurationVar(&probeTimeout, "probe-timeout", defaultProbeTimeout, "probe timeout")
	cmd.PersistentFlags().StringVar(&csiAddress, "csi-address", "/csi/csi.sock", "CSI endpoint")
	cmd.PersistentFlags().StringVar(&healthPort, "health-port", "9808", "health port")
}

func run(*cobra.Command, []string) error {
	version.LogVersion()
	logd.LogBaseLoggerSettings()

	signalHandler := ctrl.SetupSignalHandler()
	err := livenessprobe.NewServer(dtcsi.DriverName, csiAddress, healthPort, probeTimeout).Start(signalHandler)

	return errors.WithStack(err)
}
