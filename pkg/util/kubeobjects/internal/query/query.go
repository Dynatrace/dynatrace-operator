package query

import (
	"context"
	goerrors "errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Generic[T client.Object, L client.ObjectList] struct {
	Target      T
	ListTarget  L
	ToList      func(L) []T
	IsEqual     func(T, T) bool
	IsImmutable func(T, T) bool

	KubeClient client.Client
	KubeReader client.Reader
	Log        logd.Logger
}

func (c Generic[T, L]) Get(ctx context.Context, objectKey client.ObjectKey) (T, error) {
	err := c.KubeReader.Get(ctx, objectKey, c.Target)

	return c.Target, err
}

func (c Generic[T, L]) Create(ctx context.Context, object T) error {
	c.Log.Info("creating", "kind", object.GetObjectKind(), "name", object.GetName(), "namespace", object.GetNamespace())

	return errors.WithStack(c.KubeClient.Create(ctx, object))
}

func (c Generic[T, L]) Update(ctx context.Context, object T) error {
	c.Log.Info("updating", "kind", object.GetObjectKind(), "name", object.GetName(), "namespace", object.GetNamespace())

	return errors.WithStack(c.KubeClient.Update(ctx, object))
}

func (c Generic[T, L]) Delete(ctx context.Context, object T) error {
	c.Log.Info("deleting", "kind", object.GetObjectKind(), "name", object.GetName(), "namespace", object.GetNamespace())

	err := c.KubeClient.Delete(ctx, object)

	return errors.WithStack(client.IgnoreNotFound(err))
}

func (c Generic[T, L]) CreateOrUpdate(ctx context.Context, newObject T) (bool, error) {
	currentObject, err := c.Get(ctx, asNamespacedName(newObject))
	if err != nil && client.IgnoreNotFound(err) == nil {
		err = c.Create(ctx, newObject)
		if err != nil {
			return false, err
		}

		return true, nil
	} else if err != nil {
		return false, err
	}

	if c.IsEqual(currentObject, newObject) {
		c.Log.Info("update not needed, no changes detected", "kind", newObject.GetObjectKind(), "name", newObject.GetName(), "namespace", newObject.GetNamespace())

		return false, nil
	}

	if c.IsImmutable(currentObject, newObject) {
		c.Log.Info("recreation needed, immutable change detected", "kind", newObject.GetObjectKind(), "name", newObject.GetName(), "namespace", newObject.GetNamespace())

		err := c.Recreate(ctx, newObject)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	err = c.Update(ctx, newObject)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c Generic[T, L]) Recreate(ctx context.Context, object T) error {
	err := c.Delete(ctx, object)
	if err != nil {
		return err
	}

	return c.Create(ctx, object)
}

func (c Generic[T, L]) GetAllFromNamespaces(ctx context.Context, objectName string) ([]T, error) {
	c.Log.Info("querying from all namespaces", "name", objectName)

	listOps := []client.ListOption{
		client.MatchingFields{
			"metadata.name": objectName,
		},
	}

	err := c.KubeReader.List(ctx, c.ListTarget, listOps...)
	if client.IgnoreNotFound(err) != nil {
		return nil, errors.WithStack(err)
	}

	return c.ToList(c.ListTarget), err
}

func (c Generic[T, L]) CreateOrUpdateForNamespaces(ctx context.Context, object T, namespaces []corev1.Namespace) error {
	objects, err := c.GetAllFromNamespaces(ctx, object.GetName())
	if err != nil {
		return err
	}

	c.Log.Info("reconciling objects for multiple namespaces", "kind", object.GetObjectKind(),
		"name", object.GetName(), "len(namespaces)", len(namespaces))

	namespacesContainingObject := make(map[string]T, len(objects))
	for _, object := range objects {
		namespacesContainingObject[object.GetNamespace()] = object
	}

	return c.createOrUpdateForNamespaces(ctx, object, namespacesContainingObject, namespaces)
}

func (c Generic[T, L]) createOrUpdateForNamespaces(ctx context.Context, object T, namespacesContainingSecret map[string]T, namespaces []corev1.Namespace) error {
	updateCount := 0
	creationCount := 0

	var errs []error

	for _, namespace := range namespaces {
		if namespace.Status.Phase == corev1.NamespaceTerminating {
			c.Log.Info("skipping terminating namespace", "namespace", namespace.Name)

			continue
		}

		object.SetNamespace(namespace.Name)
		object.SetResourceVersion("")

		if oldObject, ok := namespacesContainingSecret[namespace.Name]; ok {
			if !c.IsEqual(oldObject, object) {
				err := c.Update(ctx, object)
				if err != nil {
					errs = append(errs, errors.WithMessagef(err, "failed to update %s %s for namespace %s", object.GetObjectKind(), object.GetName(), namespace.Name))

					continue
				}

				updateCount++
			}
		} else {
			err := c.Create(ctx, object)
			if err != nil {
				errs = append(errs, errors.WithMessagef(err, "failed to create %s %s for namespace %s", object.GetObjectKind(), object.GetName(), namespace.Name))

				continue
			}

			creationCount++
		}
	}

	c.Log.Info("reconciled objects for multiple namespaces", "kind", object.GetObjectKind(),
		"name", object.GetName(), "creationCount", creationCount, "updateCount", updateCount)

	return goerrors.Join(errs...)
}

func (c Generic[T, L]) DeleteForNamespaces(ctx context.Context, objectName string, namespaces []string) error {
	c.Log.Info("deleting objects from multiple namespaces", "name", objectName, "len(namespaces)", len(namespaces))

	errs := make([]error, 0, len(namespaces))

	for _, namespace := range namespaces {
		c.Target.SetName(objectName)
		c.Target.SetNamespace(namespace)

		err := c.Delete(ctx, c.Target)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return goerrors.Join(errs...)
}

func asNamespacedName(object client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}
}
