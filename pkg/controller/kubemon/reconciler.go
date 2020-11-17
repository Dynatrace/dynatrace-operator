package kubemon

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtpods"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtversion"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	Name                   = "kubernetes-monitoring"
	annotationTemplateHash = "internal.activegate.dynatrace.com/template-hash"
)

type Reconciler struct {
	client.Client
	scheme    *runtime.Scheme
	dtc       dtclient.Client
	log       logr.Logger
	token     *corev1.Secret
	instance  *v1alpha1.DynaKube
	apiReader client.Reader
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger, token *corev1.Secret, instance *v1alpha1.DynaKube) *Reconciler {
	return &Reconciler{
		Client:    clt,
		apiReader: apiReader,
		scheme:    scheme,
		dtc:       dtc,
		log:       log,
		token:     token,
		instance:  instance,
	}
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	result, err := dtversion.
		NewReconciler(r, r.log, r.instance, BuildLabelsFromInstance(r.instance)).
		Reconcile(request)
	if err != nil {
		return result, err
	}

	_, err = dtpullsecret.
		NewReconciler(r, r.apiReader, r.scheme, r.instance, r.dtc, r.log, r.token, r.instance.Spec.KubernetesMonitoringSpec.Image).
		Reconcile(request)
	if err != nil {
		return activegate.LogError(r.log, err, "could not reconcile Dynatrace pull secret")
	}

	_, err = customproperties.
		NewReconciler(r, r.log, Name, *r.instance.Spec.KubernetesMonitoringSpec.CustomProperties, r.scheme).
		Reconcile(request)
	if err != nil {
		return activegate.LogError(r.log, err, "could not reconcile custom properties")
	}

	err = r.manageStatefulSet(r.instance)
	if err != nil {
		return activegate.LogError(r.log, err, "could not reconcile stateful set")
	}

	if !r.instance.Spec.KubernetesMonitoringSpec.DisableActivegateUpdate {
		_, err = dtpods.
			NewReconciler(r, r.log, r.instance, BuildLabelsFromInstance(r.instance), buildImage(r.instance)).
			Reconcile(request)
		if err != nil {
			return activegate.LogError(r.log, err, "could not update pods")
		}
	}

	if r.instance.Spec.KubernetesMonitoringSpec.KubernetesAPIEndpoint != "" {
		id, err := r.addToDashboard()
		r.handleAddToDashboardResult(id, err, r.log)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) manageStatefulSet(instance *v1alpha1.DynaKube) error {
	desiredStatefulSet, err := r.buildDesiredStatefulSet(instance)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(instance, desiredStatefulSet, r.scheme); err != nil {
		return err
	}

	currentStatefulSet, err := r.createStatefulSetIfNotExists(instance, desiredStatefulSet)
	if err != nil {
		return err
	}

	err = r.updateStatefulSetIfOutdated(currentStatefulSet, desiredStatefulSet)
	if err != nil {
		return err
	}

	return r.updateInstanceStatus(instance)
}

func (r *Reconciler) buildDesiredStatefulSet(instance *v1alpha1.DynaKube) (*v1.StatefulSet, error) {
	tenantInfo, err := r.dtc.GetTenantInfo()
	if err != nil {
		return nil, err
	}

	kubeUID, err := kubesystem.GetUID(r)
	if err != nil {
		return nil, err
	}

	return newStatefulSet(*instance, tenantInfo, kubeUID), nil
}

func (r *Reconciler) createStatefulSetIfNotExists(instance *v1alpha1.DynaKube, desired *v1.StatefulSet) (*v1.StatefulSet, error) {
	currentStatefulSet, err := r.getCurrentStatefulSet(instance)
	if err != nil && k8serrors.IsNotFound(err) {
		r.log.Info("creating new stateful set for kubernetes monitoring")
		return desired, r.createStatefulSet(desired)
	}
	return currentStatefulSet, err
}

func (r *Reconciler) updateStatefulSetIfOutdated(current *v1.StatefulSet, desired *v1.StatefulSet) error {
	if hasStatefulSetChanged(current, desired) {
		r.log.Info("updating existing stateful set")
		err := r.Update(context.TODO(), desired)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) updateInstanceStatus(instance *v1alpha1.DynaKube) error {
	instance.Status.UpdatedTimestamp = metav1.NewTime(time.Now().Add(-5 * time.Minute))
	err := r.Status().Update(context.TODO(), instance)
	return err
}

func (r *Reconciler) getCurrentStatefulSet(desired *v1alpha1.DynaKube) (*v1.StatefulSet, error) {
	var currentStatefulSet *v1.StatefulSet
	err := r.Get(context.TODO(), client.ObjectKey{Name: desired.Name, Namespace: desired.Namespace}, currentStatefulSet)
	if err != nil {
		return nil, err
	}
	return currentStatefulSet, nil
}

func (r *Reconciler) createStatefulSet(desired *v1.StatefulSet) error {
	return r.Create(context.TODO(), desired)
}

func hasStatefulSetChanged(a *v1.StatefulSet, b *v1.StatefulSet) bool {
	return getTemplateHash(a) != getTemplateHash(b)
}

func getTemplateHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[annotationTemplateHash]
	}
	return ""
}
