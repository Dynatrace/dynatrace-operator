package k8scrd

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DynaKubeName    = "dynakubes.dynatrace.com"
	EdgeConnectName = "edgeconnects.dynatrace.com"
)

var log = logd.Get().WithName("operator-k8scrd")

// IsLatestVersion checks if the CRD version matches the application version and logs an error if they do not match.
func IsLatestVersion(ctx context.Context, apiReader client.Reader, crdName string) (bool, error) {
	crd := apiextensionsv1.CustomResourceDefinition{}

	appVersion := os.Getenv(k8senv.AppVersion)
	if len(appVersion) == 0 {
		return false, errors.Errorf("%s env missing, can't check for CRD version", k8senv.AppVersion)
	}

	err := apiReader.Get(ctx, types.NamespacedName{Name: crdName}, &crd)
	if err != nil {
		return false, err
	}

	crdVersion, ok := crd.Labels[k8slabel.AppVersionLabel]
	if !ok {
		return false, errors.Errorf("missing version label '%s' on %s, mismatch found", k8slabel.AppVersionLabel, crdName)
	}

	if appVersion != crdVersion {
		log.Info("outdated CRD version", "CRD name", crdName, "CRD version", crdVersion, "expected version", appVersion)

		return false, nil
	}

	return true, nil
}
