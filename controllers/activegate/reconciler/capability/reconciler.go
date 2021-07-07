package capability

import (
	"context"
	"fmt"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/internal/events"
	sts "github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/statefulset"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/tokens"
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
	dtServer        = "DT_SERVER"
	dtTenantUUID    = "DT_TENANT"
)

type Reconciler struct {
	*sts.Reconciler
	log logr.Logger
	capability.Capability
}

func NewReconciler(capability capability.Capability, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, log logr.Logger, instance *dynatracev1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider, dtc dtclient.Client) *Reconciler {
	baseReconciler := sts.NewReconciler(
		clt, apiReader, scheme, log, instance, imageVersionProvider, capability)

	baseReconciler.AddOnAfterStatefulSetCreateListener(addTenantInfo(dtc, instance.Name))

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

func setReadinessProbePort() events.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port = intstr.FromString(consts.ServiceTargetPort)
	}
}

func setCommunicationsPort(_ *dynatracev1alpha1.DynaKube) events.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				Name:          consts.ServiceTargetPort,
				ContainerPort: containerPort,
			},
		}
	}
}

func (r *Reconciler) calculateStatefulSetName() string {
	return capability.CalculateStatefulSetName(r.Capability, r.Instance.Name)
}

func addTenantInfo(dtc dtclient.Client, instanceName string) events.StatefulSetEvent {
	info, err := dtc.GetAGTenantInfo()
	if err != nil {
		return nil
	}

	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].Env = append(sts.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  dtServer,
				Value: getEndpoints(info),
			},
			corev1.EnvVar{
				Name:  dtTenantUUID,
				Value: info.TenantUUID,
			})

		const TokensSecretVolumeName = "ag-tokens-volume"
		sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: TokensSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tokens.ExtendWithAgTokensSecretSuffix(instanceName),
				},
			},
		})

		sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(sts.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      TokensSecretVolumeName,
				ReadOnly:  true,
				MountPath: "/var/lib/dynatrace/secrets",
			},
		)
	}
}

func getEndpoints(info *dtclient.TenantInfo) string {
	var endpoints strings.Builder
	endpointsLen := len(info.Endpoints)
	for i, endpoint := range info.Endpoints {
		endpoints.WriteString(endpoint.String())
		if i < endpointsLen-1 {
			endpoints.WriteRune(',')
		}
	}
	return endpoints.String()
}

func addDNSEntryPoint(instance *dynatracev1alpha1.DynaKube, moduleName string) events.StatefulSetEvent {
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
