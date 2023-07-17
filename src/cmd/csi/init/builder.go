package init

import (
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	ctrl "sigs.k8s.io/controller-runtime"
)

const use = "csi-init"

var (
	nodeId   = ""
	endpoint = ""
)

type CommandBuilder struct {
	configProvider  config.Provider
	namespace       string
}

func NewCsiInitCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) SetConfigProvider(provider config.Provider) CommandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder CommandBuilder) SetNamespace(namespace string) CommandBuilder {
	builder.namespace = namespace
	return builder
}

func (builder CommandBuilder) Build() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		RunE: builder.buildRun(),
	}

	return cmd
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		unix.Umask(0000)
		version.LogVersion()

		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		csiManager, err := createManager(builder.namespace, kubeConfig)
		if err != nil {
			return err
		}

		err = createCsiDataPath(afero.NewOsFs())
		if err != nil {
			return err
		}

		signalHandler := ctrl.SetupSignalHandler()
		access, err := metadata.NewAccess(signalHandler, dtcsi.MetadataAccessPath)
		if err != nil {
			return err
		}

		csiOptions := dtcsi.CSIOptions{
			NodeId:   nodeId,
			Endpoint: endpoint,
			RootDir:  dtcsi.DataPath,
		}

		err = metadata.NewCorrectnessChecker(csiManager.GetClient(), access, csiOptions).CorrectCSI(signalHandler)
		if err != nil {
			return err
		}
		return nil
	}
}

func createCsiDataPath(fs afero.Fs) error {
	return errors.WithStack(fs.MkdirAll(dtcsi.DataPath, 0770))
}
