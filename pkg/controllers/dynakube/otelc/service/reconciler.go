package service

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sservice"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	zipkinPortName              = "zipkin"
	zipkinPort                  = 9411
	otlpGRPCPortName            = "otlp-grpc"
	otlpGRPCPort                = 4317
	otlpHTTPPortName            = "otlp-http"
	otlpHTTPPort                = 4318
	jaegerGRPCPortName          = "jaeger-grpc"
	jaegerGRPCPort              = 14250
	jaegerThriftBinaryPortName  = "jaeger-thrift-binary"
	jaegerThriftBinaryPort      = 6832
	jaegerThriftCompactPortName = "jaeger-thrift-compact"
	jaegerThriftCompactPort     = 6831
	jaegerThriftHTTPPortName    = "jaeger-thrift-http"
	jaegerThriftHTTPPort        = 14268
	statsdPortName              = "statsd"
	statsdPort                  = 8125
	appProtocolHTTP             = "http"
	appProtocolGRPC             = "grpc"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
}

func NewReconciler(client client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:    client,
		apiReader: apiReader,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	if !dk.TelemetryIngest().IsEnabled() {
		r.removeServiceOnce(ctx, dk)

		return nil
	}

	r.removeAllServicesExcept(ctx, dk.TelemetryIngest().GetServiceName(), dk)

	return r.createOrUpdateService(ctx, dk)
}

func (r *Reconciler) removeServiceOnce(ctx context.Context, dk *dynakube.DynaKube) {
	if meta.FindStatusCondition(*dk.Conditions(), serviceConditionType) == nil {
		return
	}
	defer meta.RemoveStatusCondition(dk.Conditions(), serviceConditionType)

	r.removeAllServicesExcept(ctx, "", dk)
}

func (r *Reconciler) removeAllServicesExcept(ctx context.Context, actualServiceName string, dk *dynakube.DynaKube) {
	telemetryServiceList := &corev1.ServiceList{}

	listOps := []client.ListOption{
		client.InNamespace(dk.Namespace),
		client.MatchingLabels{
			k8slabel.AppComponentLabel: k8slabel.OtelCComponentLabel,
			k8slabel.AppCreatedByLabel: dk.Name,
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

func (r *Reconciler) createOrUpdateService(ctx context.Context, dk *dynakube.DynaKube) error {
	newService, err := r.buildService(dk)
	if err != nil {
		k8sconditions.SetServiceGenFailed(dk.Conditions(), serviceConditionType, err)

		return err
	}

	_, err = k8sservice.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newService)
	if err != nil {
		log.Info("failed to create/update telemetry service")
		k8sconditions.SetKubeAPIError(dk.Conditions(), serviceConditionType, err)

		return err
	}

	k8sconditions.SetServiceCreated(dk.Conditions(), serviceConditionType, dk.TelemetryIngest().GetServiceName())

	return nil
}

func (r *Reconciler) buildService(dk *dynakube.DynaKube) (*corev1.Service, error) {
	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.OtelCComponentLabel)
	appLabels := k8slabel.NewAppLabels(k8slabel.OtelCComponentLabel, dk.Name, k8slabel.OtelCComponentLabel, "")

	return k8sservice.Build(dk,
		dk.TelemetryIngest().GetServiceName(),
		appLabels.BuildMatchLabels(),
		buildServicePortList(dk.TelemetryIngest().GetProtocols()),
		k8sservice.SetLabels(coreLabels.BuildLabels()),
		k8sservice.SetType(corev1.ServiceTypeClusterIP),
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
					Name:        otlpGRPCPortName,
					Port:        otlpGRPCPort,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: ptr.To(appProtocolGRPC),
					TargetPort:  intstr.FromInt32(otlpGRPCPort),
				},
				corev1.ServicePort{
					Name:        otlpHTTPPortName,
					Port:        otlpHTTPPort,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: ptr.To(appProtocolHTTP),
					TargetPort:  intstr.FromInt32(otlpHTTPPort),
				})
		case otelcgen.JaegerProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:        jaegerGRPCPortName,
					Port:        jaegerGRPCPort,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: ptr.To(appProtocolGRPC),
					TargetPort:  intstr.FromInt32(jaegerGRPCPort),
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
					Name:       jaegerThriftHTTPPortName,
					Port:       jaegerThriftHTTPPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(jaegerThriftHTTPPort),
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
