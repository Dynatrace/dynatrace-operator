package otel

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	errors2 "github.com/pkg/errors"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type shutdownFn func(ctx context.Context) error

// TODO: ugly!!
var otelSecretFound = false

// Start sets up and starts all components needed for creating OpenTelemetry traces and metrics as well as some common auto-instrumentation
// It logs and swallows all errors to not prevent the application from startup.
func Start(ctx context.Context, otelServiceName string, apiReader client.Reader, webhookNamespace string) func() {
	endpoint, apiToken, err := getOtelConfig(apiReader, webhookNamespace)

	// TODO: test if OTeil is really startup in Noop Mode if secret doesn't exist, maybe separate Noop startup code from otel code
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "couldn't find OTel config secret, no OTel instrumentation available")
		return func() {}
	}

	otelResource, err := newResource(otelServiceName)
	if err != nil {
		log.Error(err, "failed to create OTel resource")
		return func() {}
	}

	_, tracerProviderShutdownFn, err := setupTraces(ctx, otelResource, endpoint, apiToken)
	if err != nil {
		log.Error(err, "failed to setup tracing infrastructure")
		return func() {}
	}

	meterProvider, meterShutdownFn, err := setupMetrics(ctx, otelResource, endpoint, apiToken)
	if err != nil {
		log.Error(err, "failed to create OTLP tracer exporter")
		return func() {}
	}

	startAutoInstrumentation(meterProvider)

	return func() {
		_ = tracerProviderShutdownFn(ctx)
		_ = meterShutdownFn(ctx)
	}
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
	return r, err
}

func getOtelConfig(apiReader client.Reader, namespace string) (string, string, error) {
	if apiReader == nil {
		return "", "", errors2.Errorf("invalid API reader")
	}

	secretName := types.NamespacedName{
		Namespace: namespace,
		Name:      otelSecretName,
	}

	query := kubeobjects.NewSecretQuery(context.Background(), nil, apiReader, log)
	secret, err := query.Get(secretName)

	if err != nil {
		return "", "", err
	}
	otelSecretFound = true

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

func shouldUseOtel() bool {
	return otelSecretFound
}

func noopShutdownFn(_ context.Context) error {
	return nil
}
