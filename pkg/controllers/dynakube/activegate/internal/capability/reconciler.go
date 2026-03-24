package capability

import (
	"context"
	"maps"
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ dynakubeReconciler = &Reconciler{}

type dynakubeReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

type Reconciler struct {
	client                     client.Client
	capability                 capability.Capability
	statefulsetReconciler      dynakubeReconciler
	customPropertiesReconciler dynakubeReconciler
	tlsSecretReconciler        dynakubeReconciler
}

func NewReconciler(clt client.Client, capability capability.Capability, statefulsetReconciler dynakubeReconciler, customPropertiesReconciler dynakubeReconciler, tlsSecretReconciler dynakubeReconciler) *Reconciler { //nolint:revive
	return &Reconciler{
		statefulsetReconciler:      statefulsetReconciler,
		customPropertiesReconciler: customPropertiesReconciler,
		tlsSecretReconciler:        tlsSecretReconciler,
		capability:                 capability,
		client:                     clt,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	err := r.customPropertiesReconciler.Reconcile(ctx, dk)
	if err != nil {
		return errors.WithStack(err)
	}

	err = r.createOrUpdateService(ctx, dk)
	if err != nil {
		return err
	}

	err = r.setAGServiceIPs(ctx, dk)
	if err != nil {
		return err
	}

	err = r.tlsSecretReconciler.Reconcile(ctx, dk)
	if err != nil {
		return err
	}

	err = r.statefulsetReconciler.Reconcile(ctx, dk)

	return errors.WithStack(err)
}

func (r *Reconciler) setAGServiceIPs(ctx context.Context, dk *dynakube.DynaKube) error {
	template := CreateService(dk)
	present := &corev1.Service{}

	// retry because a Service created by the preceding createOrUpdateService call may not be immediately visible in the API.
	return retry.OnError(retry.DefaultBackoff, k8serrors.IsNotFound, func() error {
		err := r.client.Get(ctx, client.ObjectKeyFromObject(template), present)
		if err != nil {
			return errors.WithStack(err)
		}

	dk.Status.ActiveGate.ServiceIPs = present.Spec.ClusterIPs

		return nil
	})
}

func (r *Reconciler) createOrUpdateService(ctx context.Context, dk *dynakube.DynaKube) error {
	desired := CreateService(dk)
	installed := &corev1.Service{}

	err := r.client.Get(ctx, client.ObjectKeyFromObject(desired), installed)
	if k8serrors.IsNotFound(err) {
		log.Info("creating AG service", "dk", dk.Name)

		err = controllerutil.SetControllerReference(dk, desired, r.client.Scheme())
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
		desired.ResourceVersion = installed.ResourceVersion

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
	return !maps.Equal(installedService.Labels, desiredService.Labels) ||
		!maps.Equal(installedService.Spec.Selector, desiredService.Spec.Selector)
}
