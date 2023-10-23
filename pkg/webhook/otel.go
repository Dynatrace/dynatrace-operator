package webhook

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	otelSecretName     = "dynatrace-operator-otel-config"
	otelApiEndpointKey = "endpoint"
	otelAccessTokenKey = "apiToken"
	otelBaseApiUrl     = "/api/v2/otlp"
	webhookServiceKey  = "dynatrace-webhook"

	otelMetricReadInterval = 5 * time.Second

	// these are the default values defined by OTel spec but not exported as consts in SDK packages
	otelTracesUrl  = "/v1/traces"
	otelMetricsUrl = "/v1/metrics"
)

func SetupWebhookOtel(ctx context.Context, apiReader client.Reader, webhookNamespace string) func() {
	endpoint, apiToken, err := getOtelConfig(apiReader, webhookNamespace)
	if err != nil {
		log.Error(err, "couldn't find OTel config secret, no OTel instrumentation available")
		return func() {}
	}

	otelResource, err := newResource()
	if err != nil {
		log.Error(err, "could not create OTel resource")
		return func() {}
	}

	tracerExporter, err := newOtlpTraceExporter(ctx, endpoint, apiToken)
	if err != nil {
		log.Error(err, "could not create OTLP tracer exporter")
		return func() {}
	}

	meterExporter, err := newOtlpMetricsExporter(ctx, endpoint, apiToken)
	if err != nil {
		log.Error(err, "could not create OTLP tracer exporter")
		return func() {}
	}

	tracerProvider := installTraceProvider(tracerExporter, otelResource)
	meterProvider := installMeterProvider(meterExporter, otelResource)
	return func() {
		_ = tracerProvider.Shutdown(ctx)
		_ = meterProvider.Shutdown(ctx)
	}
}

// newResource returns a resource describing this application.
func newResource() (*resource.Resource, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(webhookServiceKey),
		),
	)
	return r, err
}

func getOtelConfig(apiReader client.Reader, namespace string) (string, string, error) {
	secretName := types.NamespacedName{
		Namespace: namespace,
		Name:      otelSecretName,
	}

	query := kubeobjects.NewSecretQuery(context.Background(), nil, apiReader, log)
	secret, err := query.Get(secretName)
	if err != nil {
		return "", "", errors.WithStack(err)
	}

	endpoint, err := kubeobjects.ExtractToken(&secret, otelApiEndpointKey)
	if err != nil {
		return "", "", err
	}

	token, err := kubeobjects.ExtractToken(&secret, otelAccessTokenKey)
	if err != nil {
		return "", "", err
	}
	return endpoint, token, nil
}

func installTraceProvider(exporter sdktrace.SpanExporter, resource *resource.Resource) *sdktrace.TracerProvider {
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)

	otel.SetTracerProvider(tracerProvider)

	log.Info("OTel tracer provider installed successfully.")
	return tracerProvider
}

func newOtlpTraceExporter(ctx context.Context, endpoint string, apiToken string) (sdktrace.SpanExporter, error) {
	log.Info("setup OTel ingest", "endpoint", endpoint)
	otlpHttpClient := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath(otelBaseApiUrl+otelTracesUrl),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": "Api-Token " + apiToken,
		}))
	exporter, err := otlptrace.New(ctx, otlpHttpClient)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return exporter, nil
}

func installMeterProvider(exporter sdkmetric.Exporter, resource *resource.Resource) *sdkmetric.MeterProvider {
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(otelMetricReadInterval))),
	)

	otel.SetMeterProvider(meterProvider)
	log.Info("OTel meter provider installed successfully.")
	return meterProvider
}

func newOtlpMetricsExporter(ctx context.Context, endpoint string, apiToken string) (sdkmetric.Exporter, error) {
	var deltaTemporalitySelector = func(sdkmetric.InstrumentKind) metricdata.Temporality { return metricdata.DeltaTemporality }

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithTemporalitySelector(deltaTemporalitySelector),
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithURLPath(otelBaseApiUrl+otelMetricsUrl),
		otlpmetrichttp.WithHeaders(map[string]string{
			"Authorization": "Api-Token " + apiToken,
		}))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return exporter, nil
}
