package operator

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
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
	signalHandler            context.Context
	client                   client.Client
	namespace                string
	podName                  string
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

// TODO: This can't be stateless (so pointer receiver needs to be used), because the ctrl.SetupSignalHandler() can only be called once in a process, otherwise we get a panic. This "builder" pattern has to be refactored.
func (builder *CommandBuilder) getSignalHandler() context.Context {
	if builder.signalHandler == nil {
		builder.signalHandler = ctrl.SetupSignalHandler()
	}

	return builder.signalHandler
}

func (builder CommandBuilder) Build() *cobra.Command {
	return &cobra.Command{
		Use:          use,
		RunE:         builder.buildRun(),
		SilenceUsage: true,
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
		version.LogVersion()
		logd.LogBaseLoggerSettings()

		kubeCfg, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		builder, err = builder.setClientFromConfig(kubeCfg)
		if err != nil {
			return err
		}

		if kubesystem.IsRunLocally() {
			log.Info("running locally in debug mode")

			return builder.runLocally(kubeCfg)
		}

		return builder.runInPod(kubeCfg)
	}
}

func (builder CommandBuilder) runInPod(kubeCfg *rest.Config) error {
	operatorPod, err := pod.Get(context.TODO(), builder.client, builder.podName, builder.namespace)
	if err != nil {
		return err
	}

	isDeployedViaOlm := kubesystem.IsDeployedViaOlm(*operatorPod)
	if !isDeployedViaOlm {
		err = builder.runBootstrapper(kubeCfg)
		if err != nil {
			return err
		}
	}

	return builder.runOperatorManager(kubeCfg, isDeployedViaOlm)
}

func (builder CommandBuilder) runLocally(kubeCfg *rest.Config) error {
	err := builder.runBootstrapper(kubeCfg)
	if err != nil {
		return err
	}

	return builder.runOperatorManager(kubeCfg, false)
}

func (builder CommandBuilder) runBootstrapper(kubeCfg *rest.Config) error {
	bootstrapManager, err := builder.getBootstrapManagerProvider().CreateManager(builder.namespace, kubeCfg)
	if err != nil {
		return err
	}

	return startBootstrapperManager(bootstrapManager, builder.namespace)
}

func (builder CommandBuilder) runOperatorManager(kubeCfg *rest.Config, isDeployedViaOlm bool) error {
	operatorManager, err := builder.getOperatorManagerProvider(isDeployedViaOlm).CreateManager(builder.namespace, kubeCfg)
	if err != nil {
		return err
	}

	err = operatorManager.Start(builder.getSignalHandler())

	return errors.WithStack(err)
}

func startBootstrapperManager(bootstrapManager ctrl.Manager, namespace string) error {
	ctx, cancelFn := context.WithCancel(context.Background())

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
