package otel

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type shutdownFn func(ctx context.Context) error

// Start sets up and starts all components needed for creating OpenTelemetry traces and metrics as well as some common auto-instrumentation
// It logs and swallows all errors to not prevent the application from startup.
func Start(ctx context.Context, otelServiceName string, apiReader client.Reader, webhookNamespace string) func() {
	endpoint, apiToken, err := getOtelConfig(ctx, apiReader, webhookNamespace)

	if err != nil {
		log.Error(err, "failed to read OpenTelementry config secret")
		return setupNoopOTel()
	}

	shutdown, err := setupOtlpOTel(ctx, otelServiceName, endpoint, apiToken)
	if err != nil {
		log.Error(err, "failed to setup OTLP OpenTelementry")
		return setupNoopOTel()
	}
	return shutdown
}

func setupOtlpOTel(ctx context.Context, otelServiceName string, endpoint string, apiToken string) (func(), error) {
	otelResource, err := newResource(otelServiceName)
	if err != nil {
		return nil, err
	}

	_, tracesShutdownFn, err := setupTracesWithOtlp(ctx, otelResource, endpoint, apiToken)
	if err != nil {
		return nil, err
	}

	meterProvider, metricsShutdownFn, err := setupMetricsWithOtlp(ctx, otelResource, endpoint, apiToken)
	if err != nil {
		_ = tracesShutdownFn(ctx)
		return nil, err
	}
	startAutoInstrumentation(meterProvider)

	return func() {
		_ = tracesShutdownFn(ctx)
		_ = metricsShutdownFn(ctx)
	}, nil
}

// setupNoopOTel makes sure, that OpenTelementry is properly configured so that no subsequent usage of OpenTelementry leads to panics while keeping
// a minimal impact on runtime. Basically all collected metrics and traces get discarded right away.
func setupNoopOTel() func() {
	otel.SetMeterProvider(noop.NewMeterProvider())
	otel.SetTracerProvider(trace.NewNoopTracerProvider())
	log.Info("use Noop providers for OpenTelemetry")
	return func() {}
}

// newResource returns a resource describing this application.
func newResource(otelServiceName string) (*resource.Resource, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(otelServiceName),
		),
	)
	return r, errors.WithStack(err)
}

func getOtelConfig(ctx context.Context, apiReader client.Reader, namespace string) (string, string, error) {
	if apiReader == nil {
		return "", "", errors.Errorf("invalid API reader")
	}

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
