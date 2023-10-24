package controller_runtime

import (
	dtotel "github.com/Dynatrace/dynatrace-operator/pkg/util/otel"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ client.Writer = wrappedWriter{}

const writerSpanNamePrefix = "client.Writer"

type wrappedWriter struct {
	wrapped client.Writer
}

func (w wrappedWriter) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), writerSpanNamePrefix+".Create")
	defer span.End()

	err := w.wrapped.Create(ctx, obj, opts...)
	span.RecordError(err)

	return err
}

// Delete deletes the given obj from Kubernetes cluster.
func (w wrappedWriter) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), writerSpanNamePrefix+".Delete")
	defer span.End()

	err := w.wrapped.Delete(ctx, obj, opts...)
	span.RecordError(err)

	return err
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (w wrappedWriter) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), writerSpanNamePrefix+".Update")
	defer span.End()

	err := w.wrapped.Update(ctx, obj, opts...)
	span.RecordError(err)

	return err
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (w wrappedWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), writerSpanNamePrefix+".Patch")
	defer span.End()

	err := w.wrapped.Patch(ctx, obj, patch, opts...)
	span.RecordError(err)

	return err
}

// DeleteAllOf deletes all objects of the given type matching the given options.
func (w wrappedWriter) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), writerSpanNamePrefix+".DeleteAllOf")
	defer span.End()

	err := w.wrapped.DeleteAllOf(ctx, obj, opts...)
	span.RecordError(err)

	return err
}
