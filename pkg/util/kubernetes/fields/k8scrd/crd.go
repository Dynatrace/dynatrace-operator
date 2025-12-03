package k8scrd

import (
	"context"
	"os"

	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DynaKubeName     = "dynakubes.dynatrace.com"
	EdgeConnectName  = "edgeconnect.dynatrace.com"

	appVersionEnv      = "APP_VERSION"
	appVersionLabelKey = "app.kubernetes.io/version"
)

// CheckVersion checks if the CRD version matches the application version and logs an error if they do not match.
func CheckVersion(ctx context.Context, apiReader client.Reader, crdName string) (error) {
	crd := apiextensionsv1.CustomResourceDefinition{}

	appVersion := os.Getenv(appVersionEnv)
	if len(appVersion) == 0 {
		return errors.Errorf("%s env missing, can't check for CRD version", appVersionEnv)
	}
	
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
		log.Error(errors.Errorf("mismatching version found"), "outdated CRD version", "CRD name", crdName, "CRD version", crdVersion, "expected version", appVersion)
	}

	return nil
}