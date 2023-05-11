package validation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

const (
	errorNoResources = `No resources for istio available`
)

func noResourcesAvailable(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Spec.EnableIstio {
		enabled, err := checkIstioInstalled(dv.cfg)
		if !enabled || err != nil {
			return errorNoResources
		}
	}

	return ""
}

func checkIstioInstalled(cfg *rest.Config) (bool, error) {
	discoveryclient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}

	_, err = discoveryclient.ServerResourcesForGroupVersion(istio.IstioGVR)
	if errors.IsNotFound(err) {
		return false, nil
	}

	return err == nil, err
}
