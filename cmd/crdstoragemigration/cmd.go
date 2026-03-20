package crdstoragemigration

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/crdstoragemigration"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const use = "crd-storage-migration"

var retryFlagValue bool

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:          use,
		RunE:         run,
		SilenceUsage: true,
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&retryFlagValue, "retry", false, "Retry until completion")
}

func run(cmd *cobra.Command, args []string) error {
	version.LogVersion()

	kubeCfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	clt, err := client.New(kubeCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return err
	}

	if retryFlagValue {
		return crdstoragemigration.InitReconcile(cmd.Context(), clt, k8senv.DefaultNamespace())
	}

	return crdstoragemigration.Run(cmd.Context(), clt, k8senv.DefaultNamespace())
}
