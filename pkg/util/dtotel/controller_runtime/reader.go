package controller_runtime

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ client.Reader = wrappedReader{}

// Reader knows how to read and list Kubernetes objects.
type wrappedReader struct {
	wrapped client.Reader
}

func NewReader(wrapped client.Reader) client.Reader {
	return wrappedReader{
		wrapped: wrapped,
	}
}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (r wrappedReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	ctx, span := dtotel.StartSpan(ctx, controllerRuntimeTracer())
	defer span.End()

	err := r.wrapped.Get(ctx, key, obj, opts...)
	span.RecordError(err)

	return err
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (r wrappedReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	ctx, span := dtotel.StartSpan(ctx, controllerRuntimeTracer())
	defer span.End()

	err := r.wrapped.List(ctx, list, opts...)
	span.RecordError(err)

	return err
}
