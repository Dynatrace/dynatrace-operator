package capability

import (
	"context"
	"reflect"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ controllers.Reconciler = &Reconciler{}

type Reconciler struct {
	client                     client.Client
	capability                 capability.Capability
	statefulsetReconciler      controllers.Reconciler
	customPropertiesReconciler controllers.Reconciler
	dynakube                   *dynatracev1beta1.DynaKube
}

func NewReconciler(clt client.Client, capability capability.Capability, dynakube *dynatracev1beta1.DynaKube, statefulsetReconciler controllers.Reconciler, customPropertiesReconciler controllers.Reconciler) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		statefulsetReconciler:      statefulsetReconciler,
		customPropertiesReconciler: customPropertiesReconciler,
		capability:                 capability,
		dynakube:                   dynakube,
		client:                     clt,
	}
}

type NewReconcilerFunc = func(clt client.Client, capability capability.Capability, dynakube *dynatracev1beta1.DynaKube, statefulsetReconciler controllers.Reconciler, customPropertiesReconciler controllers.Reconciler) *Reconciler

func (r *Reconciler) Reconcile() error {
	err := r.customPropertiesReconciler.Reconcile()
	if err != nil {
		return errors.WithStack(err)
	}

	if r.dynakube.NeedsActiveGateService() {
		err = r.createOrUpdateService()
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if r.dynakube.IsStatsdActiveGateEnabled() {
		err = r.createOrUpdateEecConfigMap()
		if err != nil {
			return errors.WithStack(err)
		}
	}

	err = r.statefulsetReconciler.Reconcile()
	return errors.WithStack(err)
}

func (r *Reconciler) createOrUpdateService() error {
	desired := CreateService(r.dynakube, r.capability.ShortName())
	installed := &corev1.Service{}
	err := r.client.Get(context.TODO(), kubeobjects.Key(desired), installed)

	if k8serrors.IsNotFound(err) {
		log.Info("creating AG service", "module", r.capability.ShortName())

		err = controllerutil.SetControllerReference(r.dynakube, desired, r.client.Scheme())
		if err != nil {
			return errors.WithStack(err)
		}

		err = r.client.Create(context.TODO(), desired)
		return errors.WithStack(err)
	}

	if err != nil {
		return errors.WithStack(err)
	}

	if r.portsAreOutdated(installed, desired) || r.labelsAreOutdated(installed, desired) {
		desired.Spec.ClusterIP = installed.Spec.ClusterIP
		desired.ObjectMeta.ResourceVersion = installed.ObjectMeta.ResourceVersion
		err = r.client.Update(context.TODO(), desired)

		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) portsAreOutdated(installedService, desiredService *corev1.Service) bool {
	return !reflect.DeepEqual(installedService.Spec.Ports, desiredService.Spec.Ports)
}

func (r *Reconciler) labelsAreOutdated(installedService, desiredService *corev1.Service) bool {
	return !reflect.DeepEqual(installedService.Labels, desiredService.Labels) ||
		!reflect.DeepEqual(installedService.Spec.Selector, desiredService.Spec.Selector)
}
