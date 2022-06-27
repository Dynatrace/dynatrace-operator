package csi

import (
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csidriver "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver"
	csigc "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/gc"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	csiprovisioner "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/provisioner"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const use = "csi-driver"

type commandBuilder struct {
	configProvider  config.Provider
	managerProvider cmdManager.Provider
	namespace       string
	filesystem      afero.Fs
	csiOptions      dtcsi.CSIOptions
}

func newCsiCommandBuilder() commandBuilder {
	return commandBuilder{}
}

func (builder commandBuilder) setConfigProvider(provider config.Provider) commandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder commandBuilder) setManagerProvider(provider cmdManager.Provider) commandBuilder {
	builder.managerProvider = provider
	return builder
}

func (builder commandBuilder) setNamespace(namespace string) commandBuilder {
	builder.namespace = namespace
	return builder
}

func (builder commandBuilder) setCsiOptions(csiOptions dtcsi.CSIOptions) commandBuilder {
	builder.csiOptions = csiOptions
	return builder
}

func (builder commandBuilder) setFilesystem(filesystem afero.Fs) commandBuilder {
	builder.filesystem = filesystem
	return builder
}

func (builder commandBuilder) getFilesystem() afero.Fs {
	if builder.filesystem == nil {
		builder.filesystem = afero.NewOsFs()
	}

	return builder.filesystem
}

func (builder commandBuilder) build() *cobra.Command {
	return &cobra.Command{
		Use:  use,
		RunE: builder.buildRun(),
	}
}

func (builder commandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		csiManager, err := builder.managerProvider.CreateManager(builder.namespace, kubeConfig)
		if err != nil {
			return err
		}

		err = createCsiDataPath(builder.getFilesystem())
		if err != nil {
			return err
		}

		// TODO: make the code below testable and test it, but in another ticket because otherwise adding the other commands will take a week
		access, err := metadata.NewAccess(dtcsi.MetadataAccessPath)
		if err != nil {
			return err
		}

		err = metadata.CorrectMetadata(csiManager.GetClient(), access)
		if err != nil {
			return err
		}

		err = csidriver.NewServer(csiManager.GetClient(), builder.csiOptions, access).SetupWithManager(csiManager)
		if err != nil {
			return err
		}

		err = csiprovisioner.NewOneAgentProvisioner(csiManager, builder.csiOptions, access).SetupWithManager(csiManager)
		if err != nil {
			return err
		}

		err = csigc.NewCSIGarbageCollector(csiManager.GetClient(), builder.csiOptions, access).SetupWithManager(csiManager)
		if err != nil {
			return err
		}

		signalHandler := ctrl.SetupSignalHandler()
		err = csiManager.Start(signalHandler)
		return errors.WithStack(err)
	}
}

func createCsiDataPath(fs afero.Fs) error {
	return errors.WithStack(fs.MkdirAll(filepath.Join(dtcsi.DataPath), 0770))
}
