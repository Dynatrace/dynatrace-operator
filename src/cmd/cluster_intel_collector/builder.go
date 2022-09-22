package cluster_intel_collector

import (
	"context"
	"fmt"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	use               = "cic"
	namespaceFlagName = "namespace"
	stdoutFlagName    = "stdout"
	targetDirFlagName = "out"
)

var (
	namespaceFlagValue string
	stdoutFlagValue    bool
	targetDirFlagValue string
)

type CommandBuilder struct {
	configProvider config.Provider
}

func NewCicCommandBuilder() CommandBuilder {
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
		Long: "Pack logs and manifests useful for troubleshooting into single tarball",
		RunE: builder.buildRun(),
	}
	addFlags(cmd)
	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&namespaceFlagValue, namespaceFlagName, "dynatrace", "Specify a different Namespace.")
	cmd.PersistentFlags().BoolVar(&stdoutFlagValue, stdoutFlagName, false, "Write tarball to stdout.")
	cmd.PersistentFlags().StringVar(&targetDirFlagValue, targetDirFlagName, "/tmp/dynatrace-operator", "Target location for tarball.")
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

		k8scluster, err := builder.GetCluster(kubeConfig)
		if err != nil {
			return err
		}
		apiReader := k8scluster.GetAPIReader()

		ctx := intelCollectorContext{
			ctx:           context.TODO(),
			clientSet:     clientSet,
			apiReader:     apiReader,
			namespaceName: namespaceFlagValue,
			toStdout:      stdoutFlagValue,
			targetDir:     targetDirFlagValue,
		}

		tarball, err := newTarball(&ctx)
		if err != nil {
			return err
		}
		defer tarball.close()

		collectors := []func(*intelCollectorContext, *intelTarball) error{
			collectOperatorVersion,
			collectLogs,
			collectManifests,
		}

		for _, c := range collectors {
			if err := c(&ctx, tarball); err != nil {
				logErrorf("failed collector: %v", err)
			}
		}

		if !ctx.toStdout {
			fmt.Printf("kubectl -n %s cp %s:%s .%s\n",
				os.Getenv("POD_NAMESPACE"),
				os.Getenv("POD_NAME"),
				tarball.tarFile.Name(),
				tarball.tarFile.Name())
		}
		return nil
	}
}
