package certgen

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const use = "certgen"

var log = logd.Get().WithName("certgen")

var crdCheck bool

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:          use,
		RunE:         run,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().BoolVar(&crdCheck, "crd-check", true, "check expected crds")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	clt, err := client.New(restConfig, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return err
	}

	if crdCheck {
		if err := checkCRDs(clt); err != nil {
			return err
		}
	} else {
		log.Info("skipping crd check")
	}

	return certificates.InitReconcile(cmd.Context(), clt, k8senv.DefaultNamespace())
}

func checkCRDs(clt client.Client) error {
	gvk := latest.GroupVersion.WithKind("DynaKube")

	_, err := clt.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		log.Info("missing expected CRD", "gvk", gvk)

		return err
	}

	if installconfig.GetModules().EdgeConnect {
		gvk := v1alpha2.GroupVersion.WithKind("EdgeConnect")

		_, err = clt.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			log.Info("missing expected CRD", "gvk", gvk)

			return err
		}
	}

	return nil
}
