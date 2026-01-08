package crdcleanup

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	use                    = "crd-cleanup"
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

	return performCRDCleanup(kubeCfg, namespaceFlagValue)
}

func performCRDCleanup(kubeCfg *rest.Config, namespace string) error {
	log.Info("starting CRD storage version cleanup")

	clt, err := client.New(kubeCfg, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	ctx := context.Background()

	// Get the DynaKube CRD
	var crd apiextensionsv1.CustomResourceDefinition

	err = clt.Get(ctx, types.NamespacedName{Name: k8scrd.DynaKubeName}, &crd)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("DynaKube CRD not found, nothing to clean up")
			return nil
		}

		return errors.Wrap(err, "failed to get DynaKube CRD")
	}

	if len(crd.Status.StoredVersions) == 0 {
		log.Info("DynaKube CRD has no storage versions, skipping cleanup")
		return nil
	}

	// Get the latest storage version from the CRD
	latestVersion := getLatestStorageVersion(&crd)
	if latestVersion == "" {
		return errors.New("failed to determine latest storage version from CRD")
	}

	log.Info("latest storage version from CRD", "version", latestVersion)

	if len(crd.Status.StoredVersions) == 1 && crd.Status.StoredVersions[0] == latestVersion {
		log.Info("DynaKube CRD has single, up-to-date storage version, no cleanup needed",
			"storedVersions", crd.Status.StoredVersions)
		return nil
	}

	log.Info("DynaKube CRD has multiple storage versions, performing migration",
		"storedVersions", crd.Status.StoredVersions,
		"targetVersion", latestVersion)

	// List all DynaKube instances using unstructured to avoid version conflicts
	// We use the storage version from the CRD to ensure compatibility
	dynakubeList := &unstructured.UnstructuredList{}
	dynakubeList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "dynatrace.com",
		Version: latestVersion,
		Kind:    "DynaKubeList",
	})

	err = clt.List(ctx, dynakubeList, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return errors.Wrap(err, "failed to list DynaKube instances")
	}

	log.Info("migrating DynaKube instances to current storage version",
		"count", len(dynakubeList.Items),
		"targetVersion", latestVersion)

	for i := range dynakubeList.Items {
		dk := &dynakubeList.Items[i]
		log.Info("migrating DynaKube instance",
			"name", dk.GetName(),
			"namespace", dk.GetNamespace())

		err = clt.Update(ctx, dk)
		if err != nil {
			return errors.Wrapf(err, "failed to update DynaKube %s/%s", dk.GetNamespace(), dk.GetName())
		}
	}

	// Update CRD status to reflect the single storage version
	crd.Status.StoredVersions = []string{latestVersion}

	err = clt.Status().Update(ctx, &crd)
	if err != nil {
		return errors.Wrap(err, "failed to update DynaKube CRD status")
	}

	log.Info("successfully migrated all DynaKube instances to current storage version")

	return nil
}

// getLatestStorageVersion returns the latest storage version from the CRD.
// It looks for the version marked as storage: true in the CRD spec.
func getLatestStorageVersion(crd *apiextensionsv1.CustomResourceDefinition) string {
	for _, version := range crd.Spec.Versions {
		if version.Storage {
			return version.Name
		}
	}

	return ""
}
