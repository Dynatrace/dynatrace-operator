package activegate

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/coreReconciler"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	context context.Context
	client.Client
	Instance  *dynatracev1beta1.DynaKube
	apiReader client.Reader
	scheme    *runtime.Scheme
}

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, instance *dynatracev1beta1.DynaKube) *Reconciler {
	return &Reconciler{
		context:   ctx,
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		Instance:  instance,
	}
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	if err := r.reconcileActiveGateProxySecret(); err != nil {
		return false, err
	}

	var caps = capability.GenerateActiveGateCapabilities(r.Instance)
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
	gen := capability.NewActiveGateProxySecretGenerator(r.Client, r.apiReader, r.Instance.Namespace, log)
	if r.Instance.NeedsActiveGateProxy() {
		return gen.GenerateForDynakube(r.context, r.Instance)
	} else {
		return gen.EnsureDeleted(r.context, r.Instance)
	}
}

func (r *Reconciler) createCapability(agCapability capability.Capability) (updated bool, err error) {
	return capability.
		NewReconciler(r.Client, agCapability, coreReconciler.NewReconciler(r.Client, r.apiReader, r.scheme, r.Instance, agCapability), r.Instance).
		Reconcile()
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
			Name:      capability.BuildServiceName(r.Instance.Name, agCapability.ShortName()),
			Namespace: r.Instance.Namespace,
		},
	}
	return kubeobjects.EnsureDeleted(r.context, r.Client, &svc)
}

func (r *Reconciler) deleteStatefulset(agCapability capability.Capability) error {
	sts := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.CalculateStatefulSetName(agCapability, r.Instance.Name),
			Namespace: r.Instance.Namespace,
		},
	}
	return kubeobjects.EnsureDeleted(r.context, r.Client, &sts)
}
