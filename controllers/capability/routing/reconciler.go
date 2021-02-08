package routing

import (
	"errors"
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	module            = "msgrouter"
	StatefulSetSuffix = "-" + module
	CapabilityEnv     = "MSGrouter"
)

type ReconcileRouting struct {
	client               client.Client
	scheme               *runtime.Scheme
	dtc                  dtclient.Client
	log                  logr.Logger
	instance             *v1alpha1.DynaKube
	imageVersionProvider dtversion.ImageVersionProvider
}

func NewReconciler(clt client.Client, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *v1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider) *ReconcileRouting {
	return &ReconcileRouting{
		client:               clt,
		scheme:               scheme,
		dtc:                  dtc,
		log:                  log,
		instance:             instance,
		imageVersionProvider: imageVersionProvider,
	}
}

func (*ReconcileRouting) Reconcile() (bool, error) {
	return false, errors.New("not implemented")
}
