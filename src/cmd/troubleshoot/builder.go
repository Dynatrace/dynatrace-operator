package troubleshoot

import (
	"net/http"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	use               = "troubleshoot"
	dynakubeFlagName  = "dynakube"
	namespaceFlagName = "namespace"
)

var (
	dynakubeFlagValue  string
	namespaceFlagValue string
)

type CommandBuilder struct {
	configProvider config.Provider
}

func NewTroubleshootCommandBuilder() CommandBuilder {
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
	cmd.PersistentFlags().StringVar(&dynakubeFlagValue, dynakubeFlagName, "dynakube", "Specify a different Dynakube name.")
	cmd.PersistentFlags().StringVar(&namespaceFlagValue, namespaceFlagName, "dynatrace", "Specify a different Namespace.")
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

		k8scluster, err := builder.GetCluster(kubeConfig)
		if err != nil {
			return err
		}

		apiReader := k8scluster.GetAPIReader()

		tests := []troubleshootFunc{
			checkNamespace,
			checkDynakube,
			checkDTClusterConnection,
			checkImagePullable,
		}

		httpClient := &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		}

		troubleshootCtx := troubleshootContext{apiReader: apiReader, httpClient: httpClient, namespaceName: namespaceFlagValue, dynakubeName: dynakubeFlagValue}
		for _, test := range tests {
			if err := test(&troubleshootCtx); err != nil {
				logErrorf(err.Error())
				return nil
			}
		}
		return nil
	}
}
