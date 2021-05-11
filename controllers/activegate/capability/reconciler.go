package capability

import (
	"context"
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
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
	containerPort   = 9999
	dtDNSEntryPoint = "DT_DNS_ENTRY_POINT"
)

type Reconciler struct {
	*activegate.Reconciler
	log logr.Logger
	Capability
}

func NewReconciler(capability Capability, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *dynatracev1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider) *Reconciler {
	baseReconciler := activegate.NewReconciler(
		clt, apiReader, scheme, dtc, log, instance, imageVersionProvider,
		capability.GetProperties(), capability.GetModuleName(), capability.GetCapabilityName(), capability.GetConfiguration().ServiceAccountOwner)

	if capability.GetConfiguration().SetDnsEntryPoint {
		baseReconciler.AddOnAfterStatefulSetCreateListener(addDNSEntryPoint(instance, capability.GetModuleName()))
	}

	if capability.GetConfiguration().SetCommunicationPort {
		baseReconciler.AddOnAfterStatefulSetCreateListener(setCommunicationsPort(instance))
	}

	if capability.GetConfiguration().SetReadinessPort {
		baseReconciler.AddOnAfterStatefulSetCreateListener(setReadinessProbePort())
	}

	return &Reconciler{
		Reconciler: baseReconciler,
		log:        log,
		Capability: capability,
	}
}

func setReadinessProbePort() activegate.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port = intstr.FromString(serviceTargetPort)
	}
}

func setCommunicationsPort(_ *dynatracev1alpha1.DynaKube) activegate.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				Name:          serviceTargetPort,
				ContainerPort: containerPort,
			},
		}
	}
}

func (r *Reconciler) calculateStatefulSetName() string {
	return CalculateStatefulSetName(r.Capability, r.Instance.Name)
}

func addDNSEntryPoint(instance *dynatracev1alpha1.DynaKube, moduleName string) activegate.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].Env = append(sts.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  dtDNSEntryPoint,
				Value: buildDNSEntryPoint(instance, moduleName),
			})
	}
}

func buildDNSEntryPoint(instance *dynatracev1alpha1.DynaKube, moduleName string) string {
	return fmt.Sprintf("https://%s/communication", buildServiceHostName(instance.Name, moduleName))
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	if r.GetConfiguration().CreateService {
		update, err = r.createServiceIfNotExists()
		if update || err != nil {
			return update, errors.WithStack(err)
		}
	}

	update, err = r.Reconciler.Reconcile()
	return update, errors.WithStack(err)
}

func (r *Reconciler) createServiceIfNotExists() (bool, error) {
	service := createService(r.Instance, r.GetModuleName())

	err := r.Get(context.TODO(), client.ObjectKey{Name: service.Name, Namespace: service.Namespace}, service)
	if err != nil && k8serrors.IsNotFound(err) {
		r.log.Info("creating service", "module", r.GetModuleName())

		if err := controllerutil.SetControllerReference(r.Instance, service, r.Scheme()); err != nil {
			return false, errors.WithStack(err)
		}

		err = r.Create(context.TODO(), service)
		return true, errors.WithStack(err)
	}
	return false, errors.WithStack(err)
}
