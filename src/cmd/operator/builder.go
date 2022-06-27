package operator

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/certificates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	use = "operator"
)

type commandBuilder struct {
	configProvider           config.Provider
	bootstrapManagerProvider cmdManager.Provider
	operatorManagerProvider  cmdManager.Provider
	isDeployedViaOlm         bool
	namespace                string
	signalHandler            context.Context
}

func newOperatorCommandBuilder() commandBuilder {
	return commandBuilder{}
}

func (builder commandBuilder) setConfigProvider(provider config.Provider) commandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder commandBuilder) setOperatorManagerProvider(provider cmdManager.Provider) commandBuilder {
	builder.operatorManagerProvider = provider
	return builder
}

func (builder commandBuilder) setBootstrapManagerProvider(provider cmdManager.Provider) commandBuilder {
	builder.bootstrapManagerProvider = provider
	return builder
}

func (builder commandBuilder) setNamespace(namespace string) commandBuilder {
	builder.namespace = namespace
	return builder
}

func (builder commandBuilder) setIsDeployedViaOlm(isDeployedViaOlm bool) commandBuilder {
	builder.isDeployedViaOlm = isDeployedViaOlm
	return builder
}

func (builder commandBuilder) setSignalHandler(ctx context.Context) commandBuilder {
	builder.signalHandler = ctx
	return builder
}

func (builder commandBuilder) getSignalHandler() context.Context {
	if builder.signalHandler == nil {
		builder.signalHandler = ctrl.SetupSignalHandler()
	}

	return builder.signalHandler
}

func (builder commandBuilder) build() *cobra.Command {
	return &cobra.Command{
		Use:  use,
		RunE: builder.buildRun(),
	}
}

func (builder commandBuilder) buildRun() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kubeCfg, err := builder.configProvider.GetConfig()

		if err != nil {
			return err
		}

		if !builder.isDeployedViaOlm {
			var bootstrapManager ctrl.Manager
			bootstrapManager, err = builder.bootstrapManagerProvider.CreateManager(builder.namespace, kubeCfg)

			if err != nil {
				return err
			}

			err = runBootstrapper(bootstrapManager, builder.namespace)

			if err != nil {
				return err
			}
		}

		operatorManager, err := builder.operatorManagerProvider.CreateManager(builder.namespace, kubeCfg)

		if err != nil {
			return err
		}

		err = operatorManager.Start(builder.getSignalHandler())

		return errors.WithStack(err)
	}
}

func runBootstrapper(bootstrapManager ctrl.Manager, namespace string) error {
	ctx, cancelFn := context.WithCancel(context.TODO())
	err := certificates.AddBootstrap(bootstrapManager, namespace, cancelFn)

	if err != nil {
		return errors.WithStack(err)
	}

	err = bootstrapManager.Start(ctx)

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
