package capability

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/events"
	sts "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/statefulset"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	httpsContainerPort = 9999
	httpContainerPort  = 9998
	dtDNSEntryPoint    = "DT_DNS_ENTRY_POINT"
)

type ActiveGateCapabilityReconciler struct {
	*sts.ActiveGateStatefulSetReconciler
	capability.Capability
}

func NewActiveGateCapabilityReconciler(capability capability.Capability, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme,
	instance *dynatracev1beta1.DynaKube) *ActiveGateCapabilityReconciler {
	statefulsetReconciler := sts.NewAGStatefulSetReconciler(
		clt, apiReader, scheme, instance, capability)

	if capability.Config().SetDnsEntryPoint {
		statefulsetReconciler.AddOnAfterStatefulSetCreateListener(addDNSEntryPoint(instance, capability.ShortName()))
	}

	if capability.Config().SetCommunicationPort {
		statefulsetReconciler.AddOnAfterStatefulSetCreateListener(setCommunicationsPort(instance))
	}

	if capability.Config().SetReadinessPort {
		statefulsetReconciler.AddOnAfterStatefulSetCreateListener(setReadinessProbePort())
	}

	return &ActiveGateCapabilityReconciler{
		ActiveGateStatefulSetReconciler: statefulsetReconciler,
		Capability:                      capability,
	}
}

func setReadinessProbePort() events.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port = intstr.FromString(consts.HttpsServiceTargetPort)
	}
}

func setCommunicationsPort(_ *dynatracev1beta1.DynaKube) events.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				Name:          consts.HttpsServiceTargetPort,
				ContainerPort: httpsContainerPort,
			},
			{
				Name:          consts.HttpServiceTargetPort,
				ContainerPort: httpContainerPort,
			},
		}
	}
}

func (reconciler *ActiveGateCapabilityReconciler) calculateStatefulSetName() string {
	return capability.CalculateStatefulSetName(reconciler.Capability, reconciler.Instance.Name)
}

func addDNSEntryPoint(instance *dynatracev1beta1.DynaKube, moduleName string) events.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].Env = append(sts.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  dtDNSEntryPoint,
				Value: buildDNSEntryPoint(instance, moduleName),
			})
	}
}

func buildDNSEntryPoint(instance *dynatracev1beta1.DynaKube, moduleName string) string {
	return fmt.Sprintf("https://%s/communication", buildServiceHostName(instance.Name, moduleName))
}

func (reconciler *ActiveGateCapabilityReconciler) Reconcile() (update bool, err error) {
	if reconciler.Config().CreateService {
		update, err = reconciler.createServiceIfNotExists()
		if update || err != nil {
			return update, errors.WithStack(err)
		}
	}

	update, err = reconciler.ActiveGateStatefulSetReconciler.Reconcile()
	return update, errors.WithStack(err)
}

func (reconciler *ActiveGateCapabilityReconciler) createServiceIfNotExists() (bool, error) {
	service := createService(reconciler.Instance, reconciler.ShortName())

	err := reconciler.Get(context.TODO(), client.ObjectKey{Name: service.Name, Namespace: service.Namespace}, service)
	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating service", "module", reconciler.ShortName())
		if err := controllerutil.SetControllerReference(reconciler.Instance, service, reconciler.Scheme()); err != nil {
			return false, errors.WithStack(err)
		}

		err = reconciler.Create(context.TODO(), service)
		return true, errors.WithStack(err)
	}
	return false, errors.WithStack(err)
}
