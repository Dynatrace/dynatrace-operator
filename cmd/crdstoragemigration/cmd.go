package crdstoragemigration

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/crdstoragemigration"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	use                    = "crd-storage-migration"
	namespaceFlagName      = "namespace"
	namespaceFlagShorthand = "n"
)

var (
	namespaceFlagValue string
)

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
	cmd.PersistentFlags().StringVarP(&namespaceFlagValue, namespaceFlagName, namespaceFlagShorthand, env.DefaultNamespace(), "Specify the namespace to search for DynaKube instances.")
}

func run(cmd *cobra.Command, args []string) error {
	version.LogVersion()

	kubeCfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	clt, err := client.New(kubeCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return errors.WithStack(err)
	}

	return crdstoragemigration.Run(context.Background(), clt, clt, namespaceFlagValue)
}
