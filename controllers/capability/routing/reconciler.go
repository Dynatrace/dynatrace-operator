package routing

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	module            = "msgrouter"
	StatefulSetSuffix = "-" + module
	capabilityName    = "MSGrouter"
)

type Reconciler struct {
	*capability.Reconciler
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *v1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider, enableUpdates bool) *Reconciler {
	return &Reconciler{
		capability.NewReconciler(
			clt, apiReader, scheme, dtc, log, instance, imageVersionProvider, enableUpdates,
			&instance.Spec.RoutingSpec.CapabilityProperties, module, capabilityName, ""),
	}
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	return r.Reconciler.Reconcile()
}
