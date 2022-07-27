package capability

import (
	"context"
	"fmt"
	"reflect"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	sts "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/statefulset"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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
	Capability
}

func NewReconciler(capability Capability, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme,
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

func setReadinessProbePort() sts.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		if activeGateContainer, err := getActiveGateContainer(sts); err == nil {
			activeGateContainer.ReadinessProbe.HTTPGet.Port = intstr.FromString(HttpsServicePortName)
		} else {
			log.Error(err, "Cannot find container in the StatefulSet", "container name", ActiveGateContainerName)
		}
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

func getActiveGateContainer(sts *appsv1.StatefulSet) (*corev1.Container, error) {
	return getContainerByName(sts.Spec.Template.Spec.Containers, ActiveGateContainerName)
}

func setCommunicationsPort(dk *dynatracev1beta1.DynaKube) sts.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		activeGateContainer, err := getActiveGateContainer(sts)
		if err == nil {
			activeGateContainer.Ports = []corev1.ContainerPort{
				{
					Name:          HttpsServicePortName,
					ContainerPort: httpsContainerPort,
				},
				{
					Name:          HttpServicePortName,
					ContainerPort: httpContainerPort,
				},
			}
		} else {
			log.Info("Cannot find container in the StatefulSet", "container name", ActiveGateContainerName)
		}

		if dk.NeedsStatsd() {
			statsdContainer, err := getContainerByName(sts.Spec.Template.Spec.Containers, StatsdContainerName)
			if err == nil {
				statsdContainer.Ports = []corev1.ContainerPort{
					{
						Name:          StatsdIngestTargetPort,
						ContainerPort: StatsdIngestPort,
						Protocol:      corev1.ProtocolUDP,
					},
				}
			} else {
				log.Info("Cannot find container in the StatefulSet", "container name", StatsdContainerName)
			}
		}
	}
}

func (r *Reconciler) calculateStatefulSetName() string {
	return CalculateStatefulSetName(r.Capability, r.Instance.Name)
}

func addDNSEntryPoint(instance *dynatracev1beta1.DynaKube, moduleName string) sts.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		if activeGateContainer, err := getActiveGateContainer(sts); err == nil {
			activeGateContainer.Env = append(activeGateContainer.Env,
				corev1.EnvVar{
					Name:  dtDNSEntryPoint,
					Value: buildDNSEntryPoint(instance, moduleName),
				})
		} else {
			log.Error(err, "Cannot find container in the StatefulSet", "container name", ActiveGateContainerName)
		}
	}
}

func buildDNSEntryPoint(instance *dynatracev1beta1.DynaKube, moduleName string) string {
	return fmt.Sprintf("https://%s/communication", buildServiceHostName(instance.Name, moduleName))
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	if r.ShouldCreateService() {
		multiCapability := NewMultiCapability(r.Instance)
		update, err = r.createOrUpdateService(multiCapability.ServicePorts)
		if update || err != nil {
			return update, errors.WithStack(err)
		}
	}

	if r.Config().CreateEecRuntimeConfig {
		update, err = r.createOrUpdateEecConfigMap()
		if update || err != nil {
			return update, errors.WithStack(err)
		}
	}

	update, err = r.Reconciler.Reconcile()
	return update, errors.WithStack(err)
}

func (r *Reconciler) createOrUpdateService(desiredServicePorts AgServicePorts) (bool, error) {
	desired := createService(r.Instance, r.ShortName(), desiredServicePorts)

	installed := &corev1.Service{}
	err := r.Get(context.TODO(), kubeobjects.Key(desired), installed)
	if k8serrors.IsNotFound(err) && desiredServicePorts.HasPorts() {
		log.Info("creating AG service", "module", r.ShortName())
		if err = controllerutil.SetControllerReference(r.Instance, desired, r.Scheme()); err != nil {
			return false, errors.WithStack(err)
		}

		err = r.Create(context.TODO(), desired)
		return true, errors.WithStack(err)
	}

	if err != nil {
		return false, errors.WithStack(err)
	}

	if r.portsAreOutdated(installed, desired) || r.labelsAreOutdated(installed, desired) {
		desired.Spec.ClusterIP = installed.Spec.ClusterIP
		desired.ObjectMeta.ResourceVersion = installed.ObjectMeta.ResourceVersion

		if desiredServicePorts.HasPorts() {
			if err := r.Update(context.TODO(), desired); err != nil {
				return false, err
			}
		} else {
			if err := r.Delete(context.TODO(), desired); err != nil {
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
