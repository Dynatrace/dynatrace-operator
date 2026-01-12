package crdstoragemigration

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/crdstoragemigration"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	cmd.PersistentFlags().StringVarP(&namespaceFlagValue, namespaceFlagName, namespaceFlagShorthand, k8senv.DefaultNamespace(), "Specify the namespace to search for DynaKube instances.")
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

	return performCRDStorageMigration(clt, namespaceFlagValue)
}

func performCRDStorageMigration(clt client.Client, namespace string) error {
	ctx := context.Background()

	var crd apiextensionsv1.CustomResourceDefinition

	err := clt.Get(ctx, types.NamespacedName{Name: k8scrd.DynaKubeName}, &crd)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("DynaKube CRD not found, nothing to migrate")

			return nil
		}

		return errors.Wrap(err, "failed to get DynaKube CRD")
	}

	_, err = crdstoragemigration.PerformCRDStorageVersionMigration(ctx, clt, clt, namespace)

	return err
}
