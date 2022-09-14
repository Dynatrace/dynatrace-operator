package log_collector

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	use               = "collectlogs"
	namespaceFlagName = "namespace"
	streamFlagName    = "stream"
)

var (
	namespaceFlagValue string
	streamFlagValue    bool
)

type CommandBuilder struct {
	configProvider config.Provider
}

func NewLogCollectorCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) SetConfigProvider(provider config.Provider) CommandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder CommandBuilder) GetCluster(kubeConfig *rest.Config) (cluster.Cluster, error) {
	return cluster.New(kubeConfig, clusterOptions)
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
	cmd.PersistentFlags().StringVar(&namespaceFlagValue, namespaceFlagName, "dynatrace", "Specify a different Namespace.")
	cmd.PersistentFlags().BoolVar(&streamFlagValue, streamFlagName, false, "Stream logs.")
}

func clusterOptions(opts *cluster.Options) {
	opts.Scheme = scheme.Scheme
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		err = dynatracev1beta1.AddToScheme(scheme.Scheme)
		if err != nil {
			return err
		}

		clientSet, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			return err
		}

		if streamFlagValue {
			streamLogs(&logCollectorContext{
				ctx:           context.TODO(),
				clientSet:     clientSet,
				namespaceName: namespaceFlagValue,
				stream:        streamFlagValue,
			})
		} else {
			collectLogs(&logCollectorContext{
				ctx:           context.TODO(),
				clientSet:     clientSet,
				namespaceName: namespaceFlagValue,
				stream:        streamFlagValue,
			})
		}
		return nil
	}
}
