package activegate

import (
	"context"
	"maps"
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) deleteService(ctx context.Context, dk *dynakube.DynaKube) error {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.BuildServiceName(dk.Name),
			Namespace: dk.Namespace,
		},
	}

	return client.IgnoreNotFound(r.client.Delete(ctx, &svc))
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

func CreateService(dk *dynakube.DynaKube) *corev1.Service {
	ports := []corev1.ServicePort{
		{
			Name:       consts.HTTPSServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HTTPSServicePort,
			TargetPort: intstr.FromString(consts.HTTPSServicePortName),
		},
		{
			Name:       consts.HTTPServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HTTPServicePort,
			TargetPort: intstr.FromString(consts.HTTPServicePortName),
		},
	}

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.ActiveGateComponentLabel)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.BuildServiceName(dk.Name),
			Namespace: dk.Namespace,
			Labels:    coreLabels.BuildLabels(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: buildSelectorLabels(dk.Name),
			Ports:    ports,
		},
	}
}

func buildSelectorLabels(dynakubeName string) map[string]string {
	appLabels := k8slabel.NewAppLabels(k8slabel.ActiveGateComponentLabel, dynakubeName, "", "")

	return appLabels.BuildMatchLabels()
}
