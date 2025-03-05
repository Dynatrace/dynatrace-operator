package service

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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
	portNameOtlpGrpc            = "otlp-grpc"
	portNameOtlpHttp            = "otlp-http"
	portNameJaegerGrpc          = "jaeger-grpc"
	portNameJaegerThriftBinary  = "jaeger-thrift-binary"
	portNameJaegerThriftCompact = "jaeger-thrift-compact"
	portNameJaegerThriftHttp    = "jaeger-thrift-http"
	portNameStatsd              = "statsd"
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
	if !r.dk.TelemetryService().IsEnabled() {
		return r.removeServiceOnce(ctx)
	}

	if r.dk.TelemetryService().ServiceName != "" {
		return r.removeServiceOnce(ctx)
	}

	return r.createOrUpdateService(ctx)
}

func (r *Reconciler) removeServiceOnce(ctx context.Context) error {
	if meta.FindStatusCondition(*r.dk.Conditions(), serviceConditionType) == nil {
		return nil
	}
	defer meta.RemoveStatusCondition(r.dk.Conditions(), serviceConditionType)

	svc, err := r.buildService()
	if err != nil {
		log.Error(err, "could not build service during cleanup")

		return err
	}

	err = service.Query(r.client, r.apiReader, log).Delete(ctx, svc)
	if err != nil {
		log.Error(err, "failed to clean up extension service")

		return nil
	}

	return nil
}

func (r *Reconciler) createOrUpdateService(ctx context.Context) error {
	newService, err := r.buildService()
	if err != nil {
		conditions.SetServiceGenFailed(r.dk.Conditions(), serviceConditionType, err)

		return err
	}

	_, err = service.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newService)
	if err != nil {
		log.Info("failed to create/update otelc service")
		conditions.SetKubeApiError(r.dk.Conditions(), serviceConditionType, err)

		return err
	}

	conditions.SetServiceCreated(r.dk.Conditions(), serviceConditionType, r.dk.TelemetryService().GetName())

	return nil
}

func (r *Reconciler) buildService() (*corev1.Service, error) {
	coreLabels := labels.NewCoreLabels(r.dk.Name, labels.OtelCComponentLabel)
	// TODO: add proper version later on
	appLabels := labels.NewAppLabels(labels.OtelCComponentLabel, r.dk.Name, labels.OtelCComponentLabel, "")

	var svcPorts []corev1.ServicePort
	if r.dk.TelemetryService().IsEnabled() && r.dk.Spec.TelemetryService.ServiceName == "" {
		svcPorts = buildServicePortList(r.dk.TelemetryService().GetProtocols())
	}

	return service.Build(r.dk,
		r.dk.TelemetryService().GetName(),
		appLabels.BuildMatchLabels(),
		svcPorts,
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
				Port:       9411,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt32(9411),
			})
		case otelcgen.OtlpProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       portNameOtlpGrpc,
					Port:       4317,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(4317),
				},
				corev1.ServicePort{
					Name:       portNameOtlpHttp,
					Port:       4318,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(4318),
				})
		case otelcgen.JaegerProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       portNameJaegerGrpc,
					Port:       14250,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(14250),
				},
				corev1.ServicePort{
					Name:       portNameJaegerThriftBinary,
					Port:       6832,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(6832),
				},
				corev1.ServicePort{
					Name:       portNameJaegerThriftCompact,
					Port:       6831,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(6831),
				},
				corev1.ServicePort{
					Name:       portNameJaegerThriftHttp,
					Port:       14268,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(14268),
				})
		case otelcgen.StatsdProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       portNameStatsd,
					Port:       8125,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(8125),
				})
		default:
			log.Info("unknown telemetry service protocol ignored", "protocol", protocol)
		}
	}

	return svcPorts
}
