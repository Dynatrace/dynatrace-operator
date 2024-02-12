package capability

import (
	"context"
	"reflect"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/object"
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

func NewReconciler(clt client.Client, capability capability.Capability, dynakube *dynatracev1beta1.DynaKube, statefulsetReconciler controllers.Reconciler, customPropertiesReconciler controllers.Reconciler) controllers.Reconciler {
	return &Reconciler{
		statefulsetReconciler:      statefulsetReconciler,
		customPropertiesReconciler: customPropertiesReconciler,
		capability:                 capability,
		dynakube:                   dynakube,
		client:                     clt,
	}
}

type NewReconcilerFunc = func(clt client.Client, capability capability.Capability, dynakube *dynatracev1beta1.DynaKube, statefulsetReconciler controllers.Reconciler, customPropertiesReconciler controllers.Reconciler) controllers.Reconciler

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.customPropertiesReconciler.Reconcile(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	if r.dynakube.NeedsActiveGateService() {
		err = r.createOrUpdateService(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	err = r.statefulsetReconciler.Reconcile(ctx)

	return errors.WithStack(err)
}

func (r *Reconciler) createOrUpdateService(ctx context.Context) error {
	desired := CreateService(r.dynakube, r.capability.ShortName())
	installed := &corev1.Service{}

	err := r.client.Get(ctx, object.Key(desired), installed)
	if k8serrors.IsNotFound(err) {
		log.Info("creating AG service", "module", r.capability.ShortName())

		err = controllerutil.SetControllerReference(r.dynakube, desired, r.client.Scheme())
		if err != nil {
			return errors.WithStack(err)
		}

		err = r.client.Create(ctx, desired)

		return errors.WithStack(err)
	}

	if err != nil {
		return errors.WithStack(err)
	}

	if r.portsAreOutdated(installed, desired) || r.labelsAreOutdated(installed, desired) {
		desired.Spec.ClusterIP = installed.Spec.ClusterIP
		desired.ObjectMeta.ResourceVersion = installed.ObjectMeta.ResourceVersion

		err = r.client.Update(ctx, desired)
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
