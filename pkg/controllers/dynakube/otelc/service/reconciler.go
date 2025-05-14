package service

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	zipkinPortName              = "zipkin"
	zipkinPort                  = 9411
	otlpGrpcPortName            = "otlp-grpc"
	otlpGrpcPort                = 4317
	otlpHttpPortName            = "otlp-http"
	otlpHttpPort                = 4318
	jaegerGrpcPortName          = "jaeger-grpc"
	jaegerGrpcPort              = 14250
	jaegerThriftBinaryPortName  = "jaeger-thrift-binary"
	jaegerThriftBinaryPort      = 6832
	jaegerThriftCompactPortName = "jaeger-thrift-compact"
	jaegerThriftCompactPort     = 6831
	jaegerThriftHttpPortName    = "jaeger-thrift-http"
	jaegerThriftHttpPort        = 14268
	statsdPortName              = "statsd"
	statsdPort                  = 8125
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
				Name:       zipkinPortName,
				Port:       zipkinPort,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt32(zipkinPort),
			})
		case otelcgen.OtlpProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       otlpGrpcPortName,
					Port:       otlpGrpcPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(otlpGrpcPort),
				},
				corev1.ServicePort{
					Name:       otlpHttpPortName,
					Port:       otlpHttpPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(otlpHttpPort),
				})
		case otelcgen.JaegerProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       jaegerGrpcPortName,
					Port:       jaegerGrpcPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(jaegerGrpcPort),
				},
				corev1.ServicePort{
					Name:       jaegerThriftBinaryPortName,
					Port:       jaegerThriftBinaryPort,
					Protocol:   corev1.ProtocolUDP,
					TargetPort: intstr.FromInt32(jaegerThriftBinaryPort),
				},
				corev1.ServicePort{
					Name:       jaegerThriftCompactPortName,
					Port:       jaegerThriftCompactPort,
					Protocol:   corev1.ProtocolUDP,
					TargetPort: intstr.FromInt32(jaegerThriftCompactPort),
				},
				corev1.ServicePort{
					Name:       jaegerThriftHttpPortName,
					Port:       jaegerThriftHttpPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(jaegerThriftHttpPort),
				})
		case otelcgen.StatsdProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       statsdPortName,
					Port:       statsdPort,
					Protocol:   corev1.ProtocolUDP,
					TargetPort: intstr.FromInt32(statsdPort),
				})
		default:
			log.Info("unknown telemetry service protocol ignored", "protocol", protocol)
		}
	}

	return svcPorts
}
