package activegate

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/authtoken"
	capabilityInternal "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/tenantinfo"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	context context.Context
	client.Client
	Dynakube                          *dynatracev1beta1.DynaKube
	apiReader                         client.Reader
	scheme                            *runtime.Scheme
	authTokenReconciler               kubeobjects.Reconciler
	tenantInfoReconciler              kubeobjects.Reconciler
	newStatefulsetReconcilerFunc      statefulset.NewReconcilerFunc
	newCapabilityReconcilerFunc       capabilityInternal.NewReconcilerFunc
	newCustomPropertiesReconcilerFunc func(customPropertiesOwnerName string, customPropertiesSource *dynatracev1beta1.DynaKubeValueSource) kubeobjects.Reconciler
}

var _ kubeobjects.Reconciler = (*Reconciler)(nil)

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) *Reconciler {
	authTokenReconciler := authtoken.NewReconciler(clt, apiReader, scheme, dynakube, dtc)
	tenantInfoReconciler := tenantinfo.NewReconciler(clt, apiReader, scheme, dynakube, dtc)
	newCustomPropertiesReconcilerFunc := func(customPropertiesOwnerName string, customPropertiesSource *dynatracev1beta1.DynaKubeValueSource) kubeobjects.Reconciler {
		return customproperties.NewReconciler(clt, dynakube, customPropertiesOwnerName, scheme, customPropertiesSource)
	}

	return &Reconciler{
		context:                           ctx,
		Client:                            clt,
		apiReader:                         apiReader,
		scheme:                            scheme,
		Dynakube:                          dynakube,
		authTokenReconciler:               authTokenReconciler,
		newCustomPropertiesReconcilerFunc: newCustomPropertiesReconcilerFunc,
		newStatefulsetReconcilerFunc:      statefulset.NewReconciler,
		newCapabilityReconcilerFunc:       capabilityInternal.NewReconciler,
		tenantInfoReconciler:              tenantInfoReconciler,
	}
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	_, err = r.tenantInfoReconciler.Reconcile()
	if err != nil {
		return false, err
	}

	if r.Dynakube.UseActiveGateAuthToken() {
		_, err := r.authTokenReconciler.Reconcile()
		if err != nil {
			return false, errors.WithMessage(err, "could not reconcile Dynatrace ActiveGateAuthToken secrets")
		}
	}

	if err := r.reconcileActiveGateProxySecret(); err != nil {
		return false, err
	}

	var caps = capability.GenerateActiveGateCapabilities(r.Dynakube)
	for _, agCapability := range caps {
		if agCapability.Enabled() {
			return r.createCapability(agCapability)
		} else {
			if err := r.deleteCapability(agCapability); err != nil {
				return false, err
			}
		}
	}
	return true, err
}

func (r *Reconciler) reconcileActiveGateProxySecret() (err error) {
	gen := capabilityInternal.NewActiveGateProxySecretGenerator(r.Client, r.apiReader, r.Dynakube.Namespace, log) // TODO ?
	if r.Dynakube.NeedsActiveGateProxy() {
		return gen.GenerateForDynakube(r.context, r.Dynakube)
	} else {
		return gen.EnsureDeleted(r.context, r.Dynakube)
	}
}

func (r *Reconciler) createCapability(agCapability capability.Capability) (updated bool, err error) {
	serviceAccountOwner := agCapability.Config().ServiceAccountOwner
	if serviceAccountOwner == "" {
		serviceAccountOwner = agCapability.ShortName()
	}

	customPropertiesReconciler := r.newCustomPropertiesReconcilerFunc(serviceAccountOwner, agCapability.Properties().CustomProperties)
	statefulsetReconciler := r.newStatefulsetReconcilerFunc(r.Client, r.apiReader, r.scheme, r.Dynakube, agCapability)

	capabilityReconciler := r.newCapabilityReconcilerFunc(r.Client, agCapability, r.Dynakube, statefulsetReconciler, customPropertiesReconciler)
	return capabilityReconciler.Reconcile()
}

func (r *Reconciler) deleteCapability(agCapability capability.Capability) error {
	if err := r.deleteStatefulset(agCapability); err != nil {
		return err
	}

	if err := r.deleteService(agCapability); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) deleteService(agCapability capability.Capability) error {
	if !agCapability.ShouldCreateService() {
		return nil
	}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.BuildServiceName(r.Dynakube.Name, agCapability.ShortName()),
			Namespace: r.Dynakube.Namespace,
		},
	}
	return kubeobjects.EnsureDeleted(r.context, r.Client, &svc)
}

func (r *Reconciler) deleteStatefulset(agCapability capability.Capability) error {
	sts := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.CalculateStatefulSetName(agCapability, r.Dynakube.Name),
			Namespace: r.Dynakube.Namespace,
		},
	}
	return kubeobjects.EnsureDeleted(r.context, r.Client, &sts)
}
