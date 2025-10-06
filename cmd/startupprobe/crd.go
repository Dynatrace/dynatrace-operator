package startupprobe

import (
	"context"
	goerrors "errors"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	appVersionEnv      = "APP_VERSION"
	appVersionLabelKey = "app.kubernetes.io/version"

	dkCRDName = "dynakubes.dynatrace.com"
	ecCRDName = "edgeconnects.dynatrace.com"
)

// Why not use version.Version? (the number we "build into" the binary)
// Because that is in no way consistent with the "version" that is in the labels, which is `.Chart.AppVersion`.
// The version in the binary tends to be branch name or an actual version, while the `.Chart.AppVersion` is never a branch name.
func checkCRDVersions(ctx context.Context) error {
	appVersion := os.Getenv(appVersionEnv)

	if len(appVersion) == 0 {
		fmt.Println("APP_VERSION env missing, can't check for CRD version") //nolint

		return nil
	}

	apiReader, err := getAPIClient()
	if err != nil {
		return err
	}

	crdNames := []string{
		dkCRDName, ecCRDName,
	}

	errs := []error{}

	for _, crdName := range crdNames {
		if err := checkCRD(ctx, apiReader, crdName, appVersion); err != nil {
			errs = append(errs, err)
		}
	}

	return goerrors.Join(errs...)
}

func checkCRD(ctx context.Context, apiReader client.Reader, crdName, appVersion string) error {
	crd := apiextensionsv1.CustomResourceDefinition{}

	err := apiReader.Get(ctx, types.NamespacedName{Name: crdName}, &crd)
	if err != nil {
		return err
	}

	if crd.Labels == nil {
		return errors.Errorf("no labels on %s, mismatch found", crdName)
	}

	crdVersion, ok := crd.Labels[appVersionLabelKey]
	if !ok {
		return errors.Errorf("missing version label '%s' on %s, mismatch found", appVersionLabelKey, crdName)
	}

	if appVersion != crdVersion {
		return errors.Errorf("mismatching version found, app version %s - crd version %s", appVersion, crdVersion)
	}

	return nil
}

// TODO: This logic is used by multiple other commands, should be put into a common place
func getAPIClient() (client.Reader, error) {
	kubeConfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	k8scluster, err := cluster.New(kubeConfig, clusterOptions)
	if err != nil {
		return nil, err
	}

	return k8scluster.GetAPIReader(), nil
}

func clusterOptions(opts *cluster.Options) {
	opts.Scheme = scheme.Scheme
}
