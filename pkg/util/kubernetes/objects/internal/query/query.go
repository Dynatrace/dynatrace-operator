package query

import (
	"context"
	goerrors "errors"
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type MutateFn[T client.Object] func(T) error

type Generic[T client.Object, L client.ObjectList] struct {
	Target       T
	ListTarget   L
	ToList       func(L) []T
	IsEqual      func(T, T) bool
	MustRecreate func(T, T) bool

	Owner      client.Object
	KubeClient client.Client
	KubeReader client.Reader
	Log        logd.Logger
}

func (c Generic[T, L]) WithOwner(owner client.Object) Generic[T, L] {
	c.Owner = owner

	return c
}

func (c Generic[T, L]) Get(ctx context.Context, objectKey client.ObjectKey) (T, error) {
	err := c.KubeReader.Get(ctx, objectKey, c.Target)

	return c.Target, errors.WithStack(err)
}

func (c Generic[T, L]) Create(ctx context.Context, object T) error {
	c.log(object).Info("creating")

	err := hasher.AddAnnotation(object)
	if err != nil {
		return errors.WithStack(err)
	}

	if c.Owner != nil {
		err := controllerutil.SetControllerReference(c.Owner, object, scheme.Scheme)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return errors.WithStack(c.KubeClient.Create(ctx, object))
}

func (c Generic[T, L]) Update(ctx context.Context, object T) error {
	c.log(object).Info("updating")

	err := hasher.AddAnnotation(object)
	if err != nil {
		return errors.WithStack(err)
	}

	if c.Owner != nil {
		err := controllerutil.SetControllerReference(c.Owner, object, scheme.Scheme)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return errors.WithStack(c.KubeClient.Update(ctx, object))
}

func (c Generic[T, L]) Delete(ctx context.Context, object T, options ...client.DeleteOption) error {
	c.log(object).Info("deleting")

	err := c.KubeClient.Delete(ctx, object, options...)

	return errors.WithStack(client.IgnoreNotFound(err))
}

func mutateDeployment(d *appsv1.Deployment) func() error {
	return func() error {
		d.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}
		return nil
	}
}

func (c Generic[T, L]) CreateOrUpdate2(ctx context.Context, obj client.Object, mutate MutateFn[T]) (bool, error) {
	op, err := controllerutil.CreateOrUpdate(ctx, c.KubeClient, obj, func() error {
		return mutate(c.Target)
	})

	if err != nil {
		return false, errors.WithStack(err)
	}

	var result bool
	if op == controllerutil.OperationResultCreated {
		result = true
	}
	return result, nil
}

func (c Generic[T, L]) CreateOrUpdate(ctx context.Context, newObject T) (bool, error) {
	currentObject, err := c.Get(ctx, asNamespacedName(newObject))
	if k8serrors.IsNotFound(err) {
		err = c.Create(ctx, newObject)
		if err != nil {
			return false, err
		}

		return true, nil
	} else if err != nil {
		return false, err
	}

	err = hasher.AddAnnotation(newObject)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if c.IsEqual(currentObject, newObject) {
		c.log(newObject).Info("update not needed, no changes detected")

		return false, nil
	}

	if c.MustRecreate(currentObject, newObject) {
		c.log(newObject).Info("recreation needed, immutable change detected")

		err := c.Recreate(ctx, newObject)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	newObject.SetUID(currentObject.GetUID())
	newObject.SetResourceVersion(currentObject.GetResourceVersion())

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

	return c.ToList(c.ListTarget), errors.WithStack(client.IgnoreNotFound(err))
}

func (c Generic[T, L]) CreateOrUpdateForNamespaces(ctx context.Context, object T, namespaces []corev1.Namespace) error {
	objects, err := c.GetAllFromNamespaces(ctx, object.GetName())
	if err != nil {
		return err
	}

	c.log(object).Info("reconciling objects for multiple namespaces", "len(namespaces)", len(namespaces))

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
				object.SetUID(oldObject.GetUID())
				object.SetResourceVersion(oldObject.GetResourceVersion())

				err := c.Update(ctx, object)
				if err != nil {
					errs = append(errs, errors.WithMessagef(err, "failed to update %s %s for namespace %s", reflect.TypeOf(object), object.GetName(), namespace.Name))

					continue
				}

				updateCount++
			}
		} else {
			err := c.Create(ctx, object)
			if err != nil {
				errs = append(errs, errors.WithMessagef(err, "failed to create %s %s for namespace %s", reflect.TypeOf(object), object.GetName(), namespace.Name))

				continue
			}

			creationCount++
		}
	}

	c.log(object).Info("reconciled objects for multiple namespaces", "creationCount", creationCount, "updateCount", updateCount)

	return goerrors.Join(errs...)
}

func (c Generic[T, L]) DeleteForNamespace(ctx context.Context, objectName string, namespace string, options ...client.DeleteOption) error {
	c.Log.Info("deleting object from namespace", "name", objectName, "namespace", namespace)

	c.Target.SetName(objectName)
	c.Target.SetNamespace(namespace)

	return c.Delete(ctx, c.Target, options...)
}

func (c Generic[T, L]) DeleteForNamespaces(ctx context.Context, objectName string, namespaces []string) error {
	c.Log.Info("deleting objects from multiple namespaces", "name", objectName, "len(namespaces)", len(namespaces))

	errs := make([]error, 0, len(namespaces))

	for _, namespace := range namespaces {
		err := c.DeleteForNamespace(ctx, objectName, namespace)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return goerrors.Join(errs...)
}

func (c Generic[T, L]) log(object T) logd.Logger {
	return c.Log.WithValues("kind", reflect.TypeOf(object), "name", object.GetName(), "namespace", object.GetNamespace())
}

func asNamespacedName(object client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}
}
