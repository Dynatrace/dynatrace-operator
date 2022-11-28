package server

import (
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csidriver "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	ctrl "sigs.k8s.io/controller-runtime"
)

const use = "csi-server"

var (
	nodeId       = ""
	probeAddress = ""
	endpoint     = ""
)

type CommandBuilder struct {
	configProvider  config.Provider
	managerProvider cmdManager.Provider
	namespace       string
	filesystem      afero.Fs
	csiOptions      *dtcsi.CSIOptions
}

func NewCsiServerCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) SetConfigProvider(provider config.Provider) CommandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder CommandBuilder) setManagerProvider(provider cmdManager.Provider) CommandBuilder {
	builder.managerProvider = provider
	return builder
}

func (builder CommandBuilder) SetNamespace(namespace string) CommandBuilder {
	builder.namespace = namespace
	return builder
}

func (builder CommandBuilder) setCsiOptions(csiOptions dtcsi.CSIOptions) CommandBuilder {
	builder.csiOptions = &csiOptions
	return builder
}

func (builder CommandBuilder) setFilesystem(filesystem afero.Fs) CommandBuilder {
	builder.filesystem = filesystem
	return builder
}

func (builder CommandBuilder) getCsiOptions() dtcsi.CSIOptions {
	if builder.csiOptions == nil {
		builder.csiOptions = &dtcsi.CSIOptions{
			NodeId:   nodeId,
			Endpoint: endpoint,
			RootDir:  dtcsi.DataPath,
		}
	}

	return *builder.csiOptions
}

func (builder CommandBuilder) getManagerProvider() cmdManager.Provider {
	if builder.managerProvider == nil {
		builder.managerProvider = newCsiDriverManagerProvider(probeAddress)
	}

	return builder.managerProvider
}

func (builder CommandBuilder) getFilesystem() afero.Fs {
	if builder.filesystem == nil {
		builder.filesystem = afero.NewOsFs()
	}

	return builder.filesystem
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
	cmd.PersistentFlags().StringVar(&nodeId, "node-id", "", "node id")
	cmd.PersistentFlags().StringVar(&endpoint, "endpoint", "unix:///tmp/csi.sock", "CSI endpoint")
	cmd.PersistentFlags().StringVar(&probeAddress, "health-probe-bind-address", ":10080", "The address the probe endpoint binds to.")
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		unix.Umask(0000)
		version.LogVersion()

		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		csiManager, err := builder.getManagerProvider().CreateManager(builder.namespace, kubeConfig)
		if err != nil {
			return err
		}

		err = createCsiDataPath(builder.getFilesystem())
		if err != nil {
			return err
		}

		signalHandler := ctrl.SetupSignalHandler()
		access, err := metadata.NewAccess(signalHandler, dtcsi.MetadataAccessPath)
		if err != nil {
			return err
		}

		err = metadata.CorrectMetadata(signalHandler, csiManager.GetClient(), access)
		if err != nil {
			return err
		}

		err = csidriver.NewServer(csiManager.GetClient(), builder.getCsiOptions(), access).SetupWithManager(csiManager)
		if err != nil {
			return err
		}

		err = csiManager.Start(signalHandler)
		return errors.WithStack(err)
	}
}

func createCsiDataPath(fs afero.Fs) error {
	return errors.WithStack(fs.MkdirAll(dtcsi.DataPath, 0770))
}
