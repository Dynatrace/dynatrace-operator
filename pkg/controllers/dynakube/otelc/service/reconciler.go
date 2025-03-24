package service

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/service"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	portNameZipkin              = "zipkin"
	portZipkin                  = 9411
	portNameOtlpGrpc            = "otlp-grpc"
	portOtlpGrpc                = 4317
	portNameOtlpHttp            = "otlp-http"
	portOtlpHttp                = 4318
	portNameJaegerGrpc          = "jaeger-grpc"
	portJaegerGrpc              = 14250
	portNameJaegerThriftBinary  = "jaeger-thrift-binary"
	portJaegerThriftBinary      = 6832
	portNameJaegerThriftCompact = "jaeger-thrift-compact"
	portJaegerThriftCompact     = 6831
	portNameJaegerThriftHttp    = "jaeger-thrift-http"
	portJaegerThriftHttp        = 14268
	portNameStatsd              = "statsd"
	portStatsd                  = 8125
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

type ReconcilerBuilder func(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler

func NewReconciler(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		client:    client,
		dk:        dk,
		apiReader: apiReader,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.TelemetryIngest().IsEnabled() {
		r.removeServiceOnce(ctx)

		return nil
	}

	r.removeAllServicesExcept(ctx, r.dk.TelemetryIngest().GetServiceName())

	return r.createOrUpdateService(ctx)
}

func (r *Reconciler) removeServiceOnce(ctx context.Context) {
	if meta.FindStatusCondition(*r.dk.Conditions(), serviceConditionType) == nil {
		return
	}
	defer meta.RemoveStatusCondition(r.dk.Conditions(), serviceConditionType)

	r.removeAllServicesExcept(ctx, "")
}

func (r *Reconciler) removeAllServicesExcept(ctx context.Context, actualServiceName string) {
	telemetryServiceList := &corev1.ServiceList{}

	listOps := []client.ListOption{
		client.InNamespace(r.dk.Namespace),
		client.MatchingLabels{
			labels.AppComponentLabel: labels.OtelCComponentLabel,
			labels.AppCreatedByLabel: r.dk.Name,
		},
	}

	if err := r.apiReader.List(ctx, telemetryServiceList, listOps...); err != nil {
		log.Info("failed to list telemetry services, skipping cleanup", "error", err)

		return
	}

	for _, service := range telemetryServiceList.Items {
		if service.Name != actualServiceName {
			if err := r.client.Delete(ctx, &service); err != nil {
				log.Info("failed to clean up telemetry service", "service name", service.Name, "namespace", service.Namespace, "error", err)
			} else {
				log.Info("removed unused telemetry service", "service name", service.Name, "namespace", service.Namespace)
			}
		}
	}
}

func (r *Reconciler) createOrUpdateService(ctx context.Context) error {
	newService, err := r.buildService()
	if err != nil {
		conditions.SetServiceGenFailed(r.dk.Conditions(), serviceConditionType, err)

		return err
	}

	_, err = service.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newService)
	if err != nil {
		log.Info("failed to create/update telemetry service")
		conditions.SetKubeApiError(r.dk.Conditions(), serviceConditionType, err)

		return err
	}

	conditions.SetServiceCreated(r.dk.Conditions(), serviceConditionType, r.dk.TelemetryIngest().GetServiceName())

	return nil
}

func (r *Reconciler) buildService() (*corev1.Service, error) {
	coreLabels := labels.NewCoreLabels(r.dk.Name, labels.OtelCComponentLabel)
	// TODO: add proper version later on
	appLabels := labels.NewAppLabels(labels.OtelCComponentLabel, r.dk.Name, labels.OtelCComponentLabel, "")

	return service.Build(r.dk,
		r.dk.TelemetryIngest().GetServiceName(),
		appLabels.BuildMatchLabels(),
		buildServicePortList(r.dk.TelemetryIngest().GetProtocols()),
		service.SetLabels(coreLabels.BuildLabels()),
		service.SetType(corev1.ServiceTypeClusterIP),
	)
}

func buildServicePortList(protocols []otelcgen.Protocol) []corev1.ServicePort {
	if len(protocols) == 0 {
		return nil
	}

	svcPorts := make([]corev1.ServicePort, 0)

	for _, protocol := range protocols {
		switch protocol {
		case otelcgen.ZipkinProtocol:
			svcPorts = append(svcPorts, corev1.ServicePort{
				Name:       portNameZipkin,
				Port:       portZipkin,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt32(portZipkin),
			})
		case otelcgen.OtlpProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       portNameOtlpGrpc,
					Port:       portOtlpGrpc,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(portOtlpGrpc),
				},
				corev1.ServicePort{
					Name:       portNameOtlpHttp,
					Port:       portOtlpHttp,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(portOtlpHttp),
				})
		case otelcgen.JaegerProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       portNameJaegerGrpc,
					Port:       portJaegerGrpc,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(portJaegerGrpc),
				},
				corev1.ServicePort{
					Name:       portNameJaegerThriftBinary,
					Port:       portJaegerThriftBinary,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(portJaegerThriftBinary),
				},
				corev1.ServicePort{
					Name:       portNameJaegerThriftCompact,
					Port:       portJaegerThriftCompact,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(portJaegerThriftCompact),
				},
				corev1.ServicePort{
					Name:       portNameJaegerThriftHttp,
					Port:       portJaegerThriftHttp,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(portJaegerThriftHttp),
				})
		case otelcgen.StatsdProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       portNameStatsd,
					Port:       portStatsd,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(portStatsd),
				})
		default:
			log.Info("unknown telemetry service protocol ignored", "protocol", protocol)
		}
	}

	return svcPorts
}
