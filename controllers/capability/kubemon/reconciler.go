package kubemon

import (
	v1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceAccountOwner = "kubernetes-monitoring"

	capabilityName = "kubernetes_monitoring"

	StatefulSetSuffix = "-kubemon"
	module            = "kubemon"
)

type Reconciler struct {
	*capability.Reconciler
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *v1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider) *Reconciler {
	return &Reconciler{
		capability.NewReconciler(
			clt, apiReader, scheme, dtc, log, instance, imageVersionProvider,
			&instance.Spec.KubernetesMonitoringSpec.CapabilityProperties, module, capabilityName, serviceAccountOwner),
	}
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	return r.Reconciler.Reconcile()
}
