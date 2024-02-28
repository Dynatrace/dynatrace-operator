package init

import (
	"github.com/Dynatrace/dynatrace-operator/cmd/config"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const use = "csi-init"

var nodeId, endpoint string

type CommandBuilder struct {
	configProvider config.Provider
	namespace      string
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
		unix.Umask(dtcsi.UnixUmask)
		version.LogVersion()

		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		csiManager, err := createManager(builder.namespace, kubeConfig)
		if err != nil {
			log.Info("failed to create/configure kubernetes client, will only run non-network related corrections and checks", "err", err.Error())
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

		// new schema
		conn, err := metadata.NewDBAccess(dtcsi.MetadataAccessPath)
		if err != nil {
			return err
		}
		// new migrations
		err = conn.SchemaMigration(signalHandler)
		if err != nil {
			return err
		}

		csiOptions := dtcsi.CSIOptions{
			NodeId:   nodeId,
			Endpoint: endpoint,
			RootDir:  dtcsi.DataPath,
		}

		var apiReader client.Reader
		if csiManager != nil {
			apiReader = csiManager.GetAPIReader()
		}

		err = metadata.NewCorrectnessChecker(apiReader, access, csiOptions).CorrectCSI(signalHandler)
		if err != nil {
			return err
		}

		return nil
	}
}

func createCsiDataPath(fs afero.Fs) error {
	return errors.WithStack(fs.MkdirAll(dtcsi.DataPath, 0770))
}
