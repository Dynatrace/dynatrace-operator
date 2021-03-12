package routing

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	module            = "msgrouting"
	StatefulSetSuffix = "-" + module
	capabilityName    = "MSGrouter"
	DTDNSEntryPoint   = "DT_DNS_ENTRY_POINT"
)

type Reconciler struct {
	*capability.Reconciler
	log logr.Logger
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *dynatracev1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider, enableUpdates bool) *Reconciler {
	baseReconciler := capability.NewReconciler(
		clt, apiReader, scheme, dtc, log, instance, imageVersionProvider, enableUpdates,
		&instance.Spec.RoutingSpec.CapabilityProperties, module, capabilityName, "")
	baseReconciler.AddOnAfterStatefulSetCreateListener(addDNSEntryPoint(instance))
	baseReconciler.AddOnAfterStatefulSetCreateListener(setCommunicationsPort(instance))
	baseReconciler.AddOnAfterStatefulSetCreateListener(setLivenessProbePort(instance))
	baseReconciler.AddOnAfterStatefulSetCreateListener(setReadinessProbePort(instance))
	return &Reconciler{
		Reconciler: baseReconciler,
		log:        log,
	}
}

func setReadinessProbePort(_ *dynatracev1alpha1.DynaKube) capability.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port = intstr.FromString(serviceTargetPort)
	}
}

func setLivenessProbePort(_ *dynatracev1alpha1.DynaKube) capability.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port = intstr.FromString(serviceTargetPort)
	}
}

func setCommunicationsPort(_ *dynatracev1alpha1.DynaKube) capability.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				Name:          serviceTargetPort,
				ContainerPort: servicePort,
			},
		}
	}
}

func addDNSEntryPoint(instance *dynatracev1alpha1.DynaKube) capability.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].Env = append(sts.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  DTDNSEntryPoint,
				Value: buildDNSEntryPoint(instance),
			})
	}
}

func buildDNSEntryPoint(instance *dynatracev1alpha1.DynaKube) string {
	return fmt.Sprintf("https://%s/communication", buildServiceHostName(instance.Name, module))
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	update, err = r.createServiceIfNotExists()
	if update || err != nil {
		return update, errors.WithStack(err)
	}

	update, err = r.Reconciler.Reconcile()
	return update, errors.WithStack(err)
}

func (r *Reconciler) createServiceIfNotExists() (bool, error) {
	service := createService(r.Instance, module)
	err := r.Get(context.TODO(), client.ObjectKey{Name: service.Name, Namespace: service.Namespace}, &service)
	if err != nil && k8serrors.IsNotFound(err) {
		r.log.Info("creating service for msgrouter")
		err = r.Create(context.TODO(), &service)
		return true, errors.WithStack(err)
	}
	return false, errors.WithStack(err)
}
