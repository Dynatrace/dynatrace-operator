package activegate

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/authtoken"
	capabilityInternal "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/proxy"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/secret"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset"
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
	context                           context.Context
	client                            client.Client
	dynakube                          *dynatracev1beta1.DynaKube
	apiReader                         client.Reader
	scheme                            *runtime.Scheme
	authTokenReconciler               kubeobjects.Reconciler
	tenantInfoReconciler              kubeobjects.Reconciler
	proxyReconciler                   kubeobjects.Reconciler
	newStatefulsetReconcilerFunc      statefulset.NewReconcilerFunc
	newCapabilityReconcilerFunc       capabilityInternal.NewReconcilerFunc
	newCustomPropertiesReconcilerFunc func(customPropertiesOwnerName string, customPropertiesSource *dynatracev1beta1.DynaKubeValueSource) kubeobjects.Reconciler
}

var _ kubeobjects.Reconciler = (*Reconciler)(nil)

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) (kubeobjects.Reconciler, error) {
	tenantInfoSecret, err := createTenantInfoSecret(dtc, dynakube.AGTenantSecret(), dynakube.Namespace)
	if err != nil {
		return nil, err
	}

	authTokenReconciler := authtoken.NewReconciler(clt, apiReader, scheme, dynakube, dtc)
	tenantInfoReconciler := secret.NewReconciler(clt, apiReader, tenantInfoSecret)
	proxyReconciler := proxy.NewReconciler(clt, apiReader, dynakube)
	newCustomPropertiesReconcilerFunc := func(customPropertiesOwnerName string, customPropertiesSource *dynatracev1beta1.DynaKubeValueSource) kubeobjects.Reconciler {
		return customproperties.NewReconciler(clt, dynakube, customPropertiesOwnerName, scheme, customPropertiesSource)
	}

	return &Reconciler{
		context:                           ctx,
		client:                            clt,
		apiReader:                         apiReader,
		scheme:                            scheme,
		dynakube:                          dynakube,
		authTokenReconciler:               authTokenReconciler,
		proxyReconciler:                   proxyReconciler,
		newCustomPropertiesReconcilerFunc: newCustomPropertiesReconcilerFunc,
		newStatefulsetReconcilerFunc:      statefulset.NewReconciler,
		newCapabilityReconcilerFunc:       capabilityInternal.NewReconciler,
		tenantInfoReconciler:              tenantInfoReconciler,
	}, nil
}

func createTenantInfoSecret(dtc dtclient.Client, secretName string, namespace string) (*corev1.Secret, error) {
	tenantInfo, err := dtc.GetActiveGateTenantInfo()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	data := map[string][]byte{
		secret.TenantUuidName:             []byte(tenantInfo.UUID),
		secret.TenantTokenName:            []byte(tenantInfo.Token),
		secret.CommunicationEndpointsName: []byte(tenantInfo.Endpoints),
	}

	return kubeobjects.NewSecret(secretName, namespace, data), nil
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	_, err = r.tenantInfoReconciler.Reconcile()
	if err != nil {
		return false, err
	}

	if r.dynakube.UseActiveGateAuthToken() {
		_, err := r.authTokenReconciler.Reconcile()
		if err != nil {
			return false, errors.WithMessage(err, "could not reconcile Dynatrace ActiveGateAuthToken secrets")
		}
	}

	if _, err := r.proxyReconciler.Reconcile(); err != nil {
		return false, err
	}

	var caps = capability.GenerateActiveGateCapabilities(r.dynakube)
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

func (r *Reconciler) createCapability(agCapability capability.Capability) (updated bool, err error) {
	customPropertiesReconciler := r.newCustomPropertiesReconcilerFunc(r.dynakube.ActiveGateServiceAccountOwner(), agCapability.Properties().CustomProperties)
	statefulsetReconciler := r.newStatefulsetReconcilerFunc(r.client, r.apiReader, r.scheme, r.dynakube, agCapability)

	capabilityReconciler := r.newCapabilityReconcilerFunc(r.client, agCapability, r.dynakube, statefulsetReconciler, customPropertiesReconciler)
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
	if r.dynakube.NeedsActiveGateServicePorts() {
		return nil
	}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.BuildServiceName(r.dynakube.Name, agCapability.ShortName()),
			Namespace: r.dynakube.Namespace,
		},
	}
	return kubeobjects.Delete(r.context, r.client, &svc)
}

func (r *Reconciler) deleteStatefulset(agCapability capability.Capability) error {
	sts := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.CalculateStatefulSetName(agCapability, r.dynakube.Name),
			Namespace: r.dynakube.Namespace,
		},
	}
	return kubeobjects.Delete(r.context, r.client, &sts)
}
