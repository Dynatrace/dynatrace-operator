package routing

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	log logr.Logger
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *v1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider, enableUpdates bool) *Reconciler {
	return &Reconciler{
		Reconciler: capability.NewReconciler(
			clt, apiReader, scheme, dtc, log, instance, imageVersionProvider, enableUpdates,
			&instance.Spec.RoutingSpec.CapabilityProperties, module, capabilityName, ""),
		log: log,
	}
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	update, err = r.Reconciler.Reconcile()
	if update || err != nil {
		return update, errors.WithStack(err)
	}

	return r.createServiceIfNotExists()
}

func (r *Reconciler) createServiceIfNotExists() (bool, error) {
	service := createService(r.Instance, module)
	err := r.Get(context.TODO(), client.ObjectKey{Name: service.Name, Namespace: service.Namespace}, &service)
	if err != nil && k8serrors.IsNotFound(err) {
		r.log.Info("creating service for msgrouter")
		err = r.Create(context.TODO(), &service)
		return true, errors.WithStack(err)
	}
	return false, errors.WithStack(err)
}
