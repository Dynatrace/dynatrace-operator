// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcfg"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sservice"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	zipkinPortName              = "zipkin"
	otlpGRPCPortName            = "otlp-grpc"
	otlpHTTPPortName            = "otlp-http"
	jaegerGRPCPortName          = "jaeger-grpc"
	jaegerThriftBinaryPortName  = "jaeger-thrift-binary"
	jaegerThriftCompactPortName = "jaeger-thrift-compact"
	jaegerThriftHTTPPortName    = "jaeger-thrift-http"
	statsdPortName              = "statsd"
	appProtocolHTTP             = "http"
	appProtocolGRPC             = "grpc"

	serviceConditionType = "OTELCService"
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
	ctx, _ = logd.NewFromContext(ctx, "service")
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
	log := logd.FromContext(ctx)
	telemetryServiceList := &corev1.ServiceList{}

	listOps := []client.ListOption{
		client.InNamespace(dk.Namespace),
		client.MatchingLabels{
			k8slabel.AppComponentLabel: k8slabel.OTelColComponentLabel,
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
				log.Info("failed to clean up telemetry service", "serviceName", service.Name, "error", err)
			} else {
				log.Info("removed unused telemetry service", "serviceName", service.Name)
			}
		}
	}
}

func (r *Reconciler) createOrUpdateService(ctx context.Context, dk *dynakube.DynaKube) error {
	log := logd.FromContext(ctx)

	newService, err := r.buildService(ctx, dk)
	if err != nil {
		k8sconditions.SetServiceGenFailed(dk.Conditions(), serviceConditionType, err)

		return err
	}

	_, err = k8sservice.Query(r.client, r.apiReader).CreateOrUpdate(ctx, newService)
	if err != nil {
		log.Info("failed to create/update telemetry service")
		k8sconditions.SetKubeAPIError(dk.Conditions(), serviceConditionType, err)

		return err
	}

	k8sconditions.SetServiceCreated(dk.Conditions(), serviceConditionType, dk.TelemetryIngest().GetServiceName())

	return nil
}

func (r *Reconciler) buildService(ctx context.Context, dk *dynakube.DynaKube) (*corev1.Service, error) {
	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.OTelColComponentLabel)
	appLabels := k8slabel.NewAppLabels(k8slabel.OTelColComponentLabel, dk.Name, k8slabel.OTelColComponentLabel, "")

	return k8sservice.Build(dk,
		dk.TelemetryIngest().GetServiceName(),
		appLabels.BuildMatchLabels(),
		buildServicePortList(ctx, dk.TelemetryIngest().GetProtocols()),
		k8sservice.SetLabels(coreLabels.BuildLabels()),
		k8sservice.SetType(corev1.ServiceTypeClusterIP),
	)
}

func buildServicePortList(ctx context.Context, protocols []otelcfg.Protocol) []corev1.ServicePort {
	if len(protocols) == 0 {
		return nil
	}

	svcPorts := make([]corev1.ServicePort, 0)

	for _, protocol := range protocols {
		switch protocol {
		case otelcfg.ZipkinProtocol:
			svcPorts = append(svcPorts, corev1.ServicePort{
				Name:       zipkinPortName,
				Port:       otelcfg.ZipkinReceiverPort,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt32(otelcfg.ZipkinReceiverPort),
			})
		case otelcfg.OTLPProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:        otlpGRPCPortName,
					Port:        otelcfg.OTLPReceiverGRPCPort,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: new(appProtocolGRPC),
					TargetPort:  intstr.FromInt32(otelcfg.OTLPReceiverGRPCPort),
				},
				corev1.ServicePort{
					Name:        otlpHTTPPortName,
					Port:        otelcfg.OTLPReceiverHTTPPort,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: new(appProtocolHTTP),
					TargetPort:  intstr.FromInt32(otelcfg.OTLPReceiverHTTPPort),
				})
		case otelcfg.JaegerProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:        jaegerGRPCPortName,
					Port:        otelcfg.JaegerReceiverGRPCPort,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: new(appProtocolGRPC),
					TargetPort:  intstr.FromInt32(otelcfg.JaegerReceiverGRPCPort),
				},
				corev1.ServicePort{
					Name:       jaegerThriftBinaryPortName,
					Port:       otelcfg.JaegerReceiverThriftBinaryPort,
					Protocol:   corev1.ProtocolUDP,
					TargetPort: intstr.FromInt32(otelcfg.JaegerReceiverThriftBinaryPort),
				},
				corev1.ServicePort{
					Name:       jaegerThriftCompactPortName,
					Port:       otelcfg.JaegerReceiverThriftCompactPort,
					Protocol:   corev1.ProtocolUDP,
					TargetPort: intstr.FromInt32(otelcfg.JaegerReceiverThriftCompactPort),
				},
				corev1.ServicePort{
					Name:       jaegerThriftHTTPPortName,
					Port:       otelcfg.JaegerReceiverThriftHTTPPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(otelcfg.JaegerReceiverThriftHTTPPort),
				})
		case otelcfg.StatsdProtocol:
			svcPorts = append(svcPorts,
				corev1.ServicePort{
					Name:       statsdPortName,
					Port:       otelcfg.StatsdReceiverPort,
					Protocol:   corev1.ProtocolUDP,
					TargetPort: intstr.FromInt32(otelcfg.StatsdReceiverPort),
				})
		default:
			log := logd.FromContext(ctx)
			log.Info("unknown telemetry service protocol ignored", "protocol", protocol)
		}
	}

	return svcPorts
}
