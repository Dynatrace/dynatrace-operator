package capability

import (
	"context"
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
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
	dk                         *dynakube.DynaKube
}

func NewReconciler(clt client.Client, capability capability.Capability, dk *dynakube.DynaKube, statefulsetReconciler controllers.Reconciler, customPropertiesReconciler controllers.Reconciler) controllers.Reconciler {
	return &Reconciler{
		statefulsetReconciler:      statefulsetReconciler,
		customPropertiesReconciler: customPropertiesReconciler,
		capability:                 capability,
		dk:                         dk,
		client:                     clt,
	}
}

type NewReconcilerFunc = func(clt client.Client, capability capability.Capability, dk *dynakube.DynaKube, statefulsetReconciler controllers.Reconciler, customPropertiesReconciler controllers.Reconciler) controllers.Reconciler

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.customPropertiesReconciler.Reconcile(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	if r.dk.ActiveGate().NeedsService() {
		err = r.createOrUpdateService(ctx)
		if err != nil {
			return err
		}

		err = r.setAGServiceIPs(ctx)
		if err != nil {
			return err
		}
	} else {
		r.dk.Status.ActiveGate.ServiceIPs = []string{}
	}

	err = r.statefulsetReconciler.Reconcile(ctx)

	return errors.WithStack(err)
}

func (r *Reconciler) setAGServiceIPs(ctx context.Context) error {
	template := CreateService(r.dk, r.capability.ShortName())
	present := &corev1.Service{}

	err := r.client.Get(ctx, client.ObjectKeyFromObject(template), present)
	if err != nil {
		return errors.WithStack(err)
	}

	r.dk.Status.ActiveGate.ServiceIPs = present.Spec.ClusterIPs

	return nil
}

func (r *Reconciler) createOrUpdateService(ctx context.Context) error {
	desired := CreateService(r.dk, r.capability.ShortName())
	installed := &corev1.Service{}

	err := r.client.Get(ctx, client.ObjectKeyFromObject(desired), installed)
	if k8serrors.IsNotFound(err) {
		log.Info("creating AG service", "module", r.capability.ShortName())

		err = controllerutil.SetControllerReference(r.dk, desired, r.client.Scheme())
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
