package server

import (
	"os"

	"github.com/Dynatrace/dynatrace-operator/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csidriver "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	use = "csi-server"

	metricsBindAddress = ":8080"
)

var nodeId, endpoint string

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:          use,
		RunE:         run(),
		SilenceUsage: true,
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&nodeId, "node-id", "", "node id")
	cmd.PersistentFlags().StringVar(&endpoint, "endpoint", "unix:///tmp/csi.sock", "CSI endpoint")
}

func run() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		unix.Umask(dtcsi.UnixUmask)
		installconfig.ReadModules()
		version.LogVersion()
		logd.LogBaseLoggerSettings()

		kubeConfig, err := config.NewKubeConfigProvider().GetConfig()
		if err != nil {
			return err
		}

		csiManager, err := createManager(kubeConfig, env.DefaultNamespace())
		if err != nil {
			return err
		}

		signalHandler := ctrl.SetupSignalHandler()

		err = createCSIDataPath(afero.NewOsFs())
		if err != nil {
			return err
		}

		err = csidriver.NewServer(createCsiOptions()).SetupWithManager(csiManager)
		if err != nil {
			return err
		}

		err = csiManager.Start(signalHandler)

		return errors.WithStack(err)
	}
}

func createCSIDataPath(fs afero.Fs) error {
	return errors.WithStack(fs.MkdirAll(dtcsi.DataPath, os.ModePerm))
}

func createManager(config *rest.Config, namespace string) (manager.Manager, error) {
	options := ctrl.Options{
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		Metrics: server.Options{
			BindAddress: metricsBindAddress,
		},
		Scheme: scheme.Scheme,
	}

	mgr, err := manager.New(config, options)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return mgr, nil
}

func createCsiOptions() dtcsi.CSIOptions {
	return dtcsi.CSIOptions{
		NodeId:   nodeId,
		Endpoint: endpoint,
		RootDir:  dtcsi.DataPath,
	}
}
