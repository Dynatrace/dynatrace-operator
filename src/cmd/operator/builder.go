package operator

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	use = "operator"
)

type CommandBuilder struct {
	configProvider           config.Provider
	bootstrapManagerProvider cmdManager.Provider
	operatorManagerProvider  cmdManager.Provider
	namespace                string
	podName                  string
	signalHandler            context.Context
	client                   client.Client
}

func NewOperatorCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) SetConfigProvider(provider config.Provider) CommandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder CommandBuilder) setOperatorManagerProvider(provider cmdManager.Provider) CommandBuilder {
	builder.operatorManagerProvider = provider
	return builder
}

func (builder CommandBuilder) setBootstrapManagerProvider(provider cmdManager.Provider) CommandBuilder {
	builder.bootstrapManagerProvider = provider
	return builder
}

func (builder CommandBuilder) SetNamespace(namespace string) CommandBuilder {
	builder.namespace = namespace
	return builder
}

func (builder CommandBuilder) SetPodName(podName string) CommandBuilder {
	builder.podName = podName
	return builder
}

func (builder CommandBuilder) setSignalHandler(ctx context.Context) CommandBuilder {
	builder.signalHandler = ctx
	return builder
}

func (builder CommandBuilder) setClient(client client.Client) CommandBuilder {
	builder.client = client
	return builder
}

func (builder CommandBuilder) getOperatorManagerProvider(isDeployedByOlm bool) cmdManager.Provider {
	if builder.operatorManagerProvider == nil {
		builder.operatorManagerProvider = NewOperatorManagerProvider(isDeployedByOlm)
	}

	return builder.operatorManagerProvider
}

func (builder CommandBuilder) getBootstrapManagerProvider() cmdManager.Provider {
	if builder.bootstrapManagerProvider == nil {
		builder.bootstrapManagerProvider = NewBootstrapManagerProvider()
	}

	return builder.bootstrapManagerProvider
}

func (builder CommandBuilder) getSignalHandler() context.Context {
	if builder.signalHandler == nil {
		builder.signalHandler = ctrl.SetupSignalHandler()
	}
	return builder.signalHandler
}

func (builder CommandBuilder) Build() *cobra.Command {
	return &cobra.Command{
		Use:  use,
		RunE: builder.buildRun(),
	}
}

func (builder CommandBuilder) setClientFromConfig(kubeCfg *rest.Config) (CommandBuilder, error) {
	if builder.client == nil {
		clt, err := client.New(kubeCfg, client.Options{})
		if err != nil {
			return builder, err
		}
		return builder.setClient(clt), nil
	}
	return builder, nil
}

func (builder CommandBuilder) buildRun() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kubeCfg, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		builder, err = builder.setClientFromConfig(kubeCfg)
		if err != nil {
			return err
		}

		isDeployedViaOlm := false

		if os.Getenv("RUN_LOCAL") != "true" {
			operatorPod, err := kubeobjects.GetPod(context.TODO(), builder.client, builder.podName, builder.namespace)
			if err != nil {
				return err
			}

			isDeployedViaOlm = kubesystem.IsDeployedViaOlm(*operatorPod)
			if !isDeployedViaOlm {
				var bootstrapManager ctrl.Manager
				bootstrapManager, err = builder.getBootstrapManagerProvider().CreateManager(builder.namespace, kubeCfg)

				if err != nil {
					return err
				}

				err = runBootstrapper(bootstrapManager, builder.namespace)

				if err != nil {
					return err
				}
			}
		}

		operatorManager, err := builder.getOperatorManagerProvider(isDeployedViaOlm).CreateManager(builder.namespace, kubeCfg)

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
