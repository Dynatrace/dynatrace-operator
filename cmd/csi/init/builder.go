package init

import (
	"github.com/Dynatrace/dynatrace-operator/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
		logd.LogBaseLoggerSettings()

		err := createCsiDataPath(afero.NewOsFs())
		if err != nil {
			return err
		}

		signalHandler := ctrl.SetupSignalHandler()

		csiOptions := dtcsi.CSIOptions{
			NodeId:   nodeId,
			Endpoint: endpoint,
			RootDir:  dtcsi.DataPath,
		}

		managerOptions := ctrl.Options{
			Cache: cache.Options{
				DefaultNamespaces: map[string]cache.Config{
					env.DefaultNamespace(): {},
				},
			},
			Scheme: scheme.Scheme,
		}

		kubeconfig, err := config.NewKubeConfigProvider().GetConfig()
		if err != nil {
			return errors.WithStack(err)
		}

		mgr, err := manager.New(kubeconfig, managerOptions)
		if err != nil {
			return errors.WithStack(err)
		}

		err = metadata.NewCorrectnessChecker(mgr.GetAPIReader(), csiOptions).CorrectCSI(signalHandler)
		if err != nil {
			return err
		}

		return nil
	}
}

func createCsiDataPath(fs afero.Fs) error {
	return errors.WithStack(fs.MkdirAll(dtcsi.DataPath, 0770))
}
