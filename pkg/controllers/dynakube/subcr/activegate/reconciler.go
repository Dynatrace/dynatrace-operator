package activegate

import (
	"context"

	activegatev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/activegate"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/object"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	scheme    *runtime.Scheme

	dynakube *dynatracev1beta1.DynaKube
}

func NewReconciler(client client.Client,
	apiReader client.Reader,
	scheme *runtime.Scheme,
	dynakube *dynatracev1beta1.DynaKube) *Reconciler {
	return &Reconciler{
		client:    client,
		apiReader: apiReader,
		scheme:    scheme,
		dynakube:  dynakube,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	desiredObject := activegatev1alpha1.FromDynakube(r.dynakube)
	if err := controllerutil.SetControllerReference(r.dynakube, desiredObject, r.scheme); err != nil {
		return errors.WithStack(err)
	}

	oldObject, err := r.GetActiveGate(desiredObject.Name, desiredObject.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if !r.dynakube.NeedsActiveGate() {
		if oldObject == nil {
			return nil
		}

		controllerReference := metav1.GetControllerOf(oldObject)
		if controllerReference != nil && controllerReference.UID == r.dynakube.UID {
			log.Info("deleting old ActiveGate CR", "name", oldObject.Name, "namespace", oldObject.Namespace)

			return object.Delete(ctx, r.client, desiredObject)
		}

		return nil
	}

	if !r.objectChanged(oldObject, desiredObject) {
		log.Info("ActiveGate CR is up to date", "name", desiredObject.Name, "namespace", desiredObject.Namespace)

		return nil
	}

	log.Info("updating ActiveGate CR", "name", desiredObject.Name, "namespace", desiredObject.Namespace)

	if oldObject == nil {
		return r.client.Create(ctx, desiredObject)
	}

	oldObject.Annotations = desiredObject.Annotations
	oldObject.Spec = desiredObject.Spec

	return r.client.Update(ctx, oldObject)
}

func (r *Reconciler) GetActiveGate(name, namespace string) (*activegatev1alpha1.ActiveGate, error) {
	var activeGate activegatev1alpha1.ActiveGate

	err := r.client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: namespace}, &activeGate)
	if err != nil {
		return nil, err
	}

	return &activeGate, nil
}

func (r *Reconciler) objectChanged(old *activegatev1alpha1.ActiveGate, desired *activegatev1alpha1.ActiveGate) bool {
	if old == nil {
		return true
	}

	if old.Annotations == nil {
		old.Annotations = make(map[string]string)
	}

	changed, err := hasher.IsDifferent(old.Annotations, desired.Annotations)
	if err != nil {
		log.Error(err, "failed to compare ActiveGate CRs annotations")

		return true
	} else if changed {
		return true
	}

	changed, err = hasher.IsDifferent(old.Spec, desired.Spec)
	if err != nil {
		log.Error(err, "failed to compare ActiveGate CRs spec")

		return true
	}

	return changed
}
