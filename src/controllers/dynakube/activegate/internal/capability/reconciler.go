package capability

import (
	"context"
	"reflect"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ kubeobjects.Reconciler = &Reconciler{}

type Reconciler struct {
	client                     client.Client
	capability                 capability.Capability
	statefulsetReconciler      kubeobjects.Reconciler
	customPropertiesReconciler kubeobjects.Reconciler
	dynakube                   *dynatracev1beta1.DynaKube
}

func NewReconciler(clt client.Client, capability capability.Capability, dynakube *dynatracev1beta1.DynaKube, statefulsetReconciler kubeobjects.Reconciler, customPropertiesReconciler kubeobjects.Reconciler) *Reconciler {

	return &Reconciler{
		statefulsetReconciler:      statefulsetReconciler,
		customPropertiesReconciler: customPropertiesReconciler,
		capability:                 capability,
		dynakube:                   dynakube,
		client:                     clt,
	}
}

type NewReconcilerFunc = func(clt client.Client, capability capability.Capability, dynakube *dynatracev1beta1.DynaKube, statefulsetReconciler kubeobjects.Reconciler, customPropertiesReconciler kubeobjects.Reconciler) *Reconciler

func (r *Reconciler) Reconcile() (update bool, err error) {
	_, err = r.customPropertiesReconciler.Reconcile()
	if err != nil {
		return update, errors.WithStack(err)
	}

	if r.capability.ShouldCreateService() {
		// TODO: MutliCapability shouldn't be used here - it may be as well one of deprecated Capabilities: Kubemon or Routing
		multiCapability := capability.NewMultiCapability(r.dynakube)
		update, err = r.createOrUpdateService(multiCapability.ServicePorts)
		if update || err != nil {
			return update, errors.WithStack(err)
		}
	}

	if r.capability.Config().CreateEecRuntimeConfig {
		update, err = r.createOrUpdateEecConfigMap()
		if update || err != nil {
			return update, errors.WithStack(err)
		}
	}

	update, err = r.statefulsetReconciler.Reconcile()
	return update, errors.WithStack(err)
}

func (r *Reconciler) createOrUpdateService(desiredServicePorts capability.AgServicePorts) (bool, error) {
	desired := CreateService(r.dynakube, r.capability.ShortName(), desiredServicePorts)

	installed := &corev1.Service{}
	err := r.client.Get(context.TODO(), kubeobjects.Key(desired), installed)
	if k8serrors.IsNotFound(err) && desiredServicePorts.HasPorts() {
		log.Info("creating AG service", "module", r.capability.ShortName())
		if err = controllerutil.SetControllerReference(r.dynakube, desired, r.client.Scheme()); err != nil {
			return false, errors.WithStack(err)
		}

		err = r.client.Create(context.TODO(), desired)
		return true, errors.WithStack(err)
	}

	if err != nil {
		return false, errors.WithStack(err)
	}

	if r.portsAreOutdated(installed, desired) || r.labelsAreOutdated(installed, desired) {
		desired.Spec.ClusterIP = installed.Spec.ClusterIP
		desired.ObjectMeta.ResourceVersion = installed.ObjectMeta.ResourceVersion

		if desiredServicePorts.HasPorts() {
			if err := r.client.Update(context.TODO(), desired); err != nil {
				return false, err
			}
		} else {
			if err := r.client.Delete(context.TODO(), desired); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (r *Reconciler) portsAreOutdated(installedService, desiredService *corev1.Service) bool {
	return !reflect.DeepEqual(installedService.Spec.Ports, desiredService.Spec.Ports)
}

func (r *Reconciler) labelsAreOutdated(installedService, desiredService *corev1.Service) bool {
	return !reflect.DeepEqual(installedService.Labels, desiredService.Labels) ||
		!reflect.DeepEqual(installedService.Spec.Selector, desiredService.Spec.Selector)
}
