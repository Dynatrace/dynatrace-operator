package controller_runtime

import (
	dtotel "github.com/Dynatrace/dynatrace-operator/pkg/util/otel"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ client.Client = wrappedClient{}

const clientSpanNamePrefix = "client.Client"

// Reader knows how to read and list Kubernetes objects.
type wrappedClient struct {
	wrapped client.Client
}

func NewClient(wrapped client.Client) client.Client {
	return wrappedClient{
		wrapped: wrapped,
	}
}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (c wrappedClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), clientSpanNamePrefix+".Get")
	defer span.End()

	err := c.wrapped.Get(ctx, key, obj, opts...)
	span.RecordError(err)

	return err
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (c wrappedClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), clientSpanNamePrefix+".List")
	defer span.End()

	err := c.wrapped.List(ctx, list, opts...)
	span.RecordError(err)

	return err
}

func (c wrappedClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), clientSpanNamePrefix+".Create")
	defer span.End()

	err := c.wrapped.Create(ctx, obj, opts...)
	span.RecordError(err)

	return err
}

// Delete deletes the given obj from Kubernetes cluster.
func (c wrappedClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), clientSpanNamePrefix+".Delete")
	defer span.End()

	err := c.wrapped.Delete(ctx, obj, opts...)
	span.RecordError(err)

	return err
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c wrappedClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), clientSpanNamePrefix+".Update")
	defer span.End()

	err := c.wrapped.Update(ctx, obj, opts...)
	span.RecordError(err)

	return err
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c wrappedClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), clientSpanNamePrefix+".Patch")
	defer span.End()

	err := c.wrapped.Patch(ctx, obj, patch, opts...)
	span.RecordError(err)

	return err
}

// DeleteAllOf deletes all objects of the given type matching the given options.
func (c wrappedClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	ctx, span := dtotel.StartSpan(ctx, otel.Tracer(otelTracerName), clientSpanNamePrefix+".DeleteAllOf")
	defer span.End()

	err := c.wrapped.DeleteAllOf(ctx, obj, opts...)
	span.RecordError(err)

	return err
}

// Scheme returns the scheme this client is using.
func (c wrappedClient) Scheme() *runtime.Scheme {
	return c.wrapped.Scheme()
}

// RESTMapper returns the rest this client is using.
func (c wrappedClient) RESTMapper() meta.RESTMapper {
	return c.wrapped.RESTMapper()
}

// GroupVersionKindFor returns the GroupVersionKind for the given object.
func (c wrappedClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return c.wrapped.GroupVersionKindFor(obj)
}

// IsObjectNamespaced returns true if the GroupVersionKind of the object is namespaced.
func (c wrappedClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return c.wrapped.IsObjectNamespaced(obj)
}

func (c wrappedClient) Status() client.SubResourceWriter {
	return c.wrapped.Status()
}

func (c wrappedClient) SubResource(subResource string) client.SubResourceClient {
	return c.wrapped.SubResource(subResource)
}
