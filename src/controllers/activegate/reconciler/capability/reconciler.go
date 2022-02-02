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

type Reconciler struct {
	*sts.Reconciler
	capability.Capability
}

func NewReconciler(capability capability.Capability, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme,
	instance *dynatracev1beta1.DynaKube) *Reconciler {
	baseReconciler := sts.NewReconciler(
		clt, apiReader, scheme, instance, capability)

	if capability.Config().SetDnsEntryPoint {
		baseReconciler.AddOnAfterStatefulSetCreateListener(addDNSEntryPoint(instance, capability.ShortName()))
	}

	if capability.Config().SetCommunicationPort {
		baseReconciler.AddOnAfterStatefulSetCreateListener(setCommunicationsPort(instance))
	}

	if capability.Config().SetReadinessPort {
		baseReconciler.AddOnAfterStatefulSetCreateListener(setReadinessProbePort())
	}

	return &Reconciler{
		Reconciler: baseReconciler,
		Capability: capability,
	}
}

func setReadinessProbePort() events.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port = intstr.FromString(consts.HttpsServicePortName)
	}
}

func getContainerByName(containers []corev1.Container, containerName string) (*corev1.Container, error) {
	for i := range containers {
		if containers[i].Name == containerName {
			return &containers[i], nil
		}
	}
	return nil, errors.Errorf(`Cannot find container "%s" in the provided slice (len %d)`,
		containerName, len(containers),
	)
}

func setCommunicationsPort(dk *dynatracev1beta1.DynaKube) events.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		{
			activeGateContainer, err := getContainerByName(sts.Spec.Template.Spec.Containers, consts.ActiveGateContainerName)
			if err == nil {
				activeGateContainer.Ports = []corev1.ContainerPort{
					{
						Name:          consts.HttpsServicePortName,
						ContainerPort: httpsContainerPort,
					},
					{
						Name:          consts.HttpServicePortName,
						ContainerPort: httpContainerPort,
					},
				}
			}
			// TODO How to report an error?
		}
		if dk.FeatureEnableStatsDIngest() {
			statsdContainer, err := getContainerByName(sts.Spec.Template.Spec.Containers, consts.StatsDContainerName)
			if err == nil {
				statsdContainer.Ports = []corev1.ContainerPort{
					{
						Name:          consts.StatsDIngestTargetPort,
						ContainerPort: consts.StatsDIngestPort,
						Protocol:      corev1.ProtocolUDP,
					},
				}
			}
			// TODO How to report error?
		}
	}
}

func (r *Reconciler) calculateStatefulSetName() string {
	return capability.CalculateStatefulSetName(r.Capability, r.Instance.Name)
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

func (r *Reconciler) Reconcile() (update bool, err error) {
	if r.Config().CreateService {
		update, err = r.createServiceIfNotExists()
		if update || err != nil {
			return update, errors.WithStack(err)
		}

		update, err = r.updateServiceIfOutdated()
		if update || err != nil {
			return update, errors.WithStack(err)
		}
	}

	update, err = r.Reconciler.Reconcile()
	return update, errors.WithStack(err)
}

func (r *Reconciler) createServiceIfNotExists() (bool, error) {
	service := createService(r.Instance, r.ShortName())

	err := r.Get(context.TODO(), client.ObjectKey{Name: service.Name, Namespace: service.Namespace}, service)
	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating service", "module", r.ShortName())
		if err := controllerutil.SetControllerReference(r.Instance, service, r.Scheme()); err != nil {
			return false, errors.WithStack(err)
		}

		err = r.Create(context.TODO(), service)
		return true, errors.WithStack(err)
	}
	return false, errors.WithStack(err)
}

func (r *Reconciler) updateServiceIfOutdated() (bool, error) {
	desiredService := createService(r.Instance, r.ShortName())
	installedService := &corev1.Service{}

	err := r.Get(context.TODO(), client.ObjectKey{Name: desiredService.Name, Namespace: desiredService.Namespace}, installedService)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if r.isOutdated(installedService, desiredService) {
		desiredService.Spec.ClusterIP = installedService.Spec.ClusterIP
		desiredService.ObjectMeta.ResourceVersion = installedService.ObjectMeta.ResourceVersion
		updateErr := r.updateService(desiredService)
		if updateErr != nil {
			return false, updateErr
		}
		return true, nil
	}
	return false, nil
}

func (r *Reconciler) isOutdated(installedService, desiredService *corev1.Service) bool {
	return !dynatracev1beta1.IsInternalFlagsEqual(installedService, desiredService)
}

func (r *Reconciler) updateService(service *corev1.Service) error {
	return r.Update(context.TODO(), service)
}
