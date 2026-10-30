// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sobject

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Erroto signal that an immutable field of the existing object has to change, so the object must be deleted and created again.
var ErrRecreate = errors.New("object has to be recreated")

// RetryCreateOrUpdate is a thin wrapper around [controllerutil.CreateOrUpdate] that retries on conflict errors to prevent redoing a whole reconcile.
// If the mutate function returns [ErrRecreate], the existing object is deleted and created again once.
// Prefer this for external resources, i.e. ones that are not managed by the Dynatrace Operator.
func RetryCreateOrUpdate(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) error {
	gvks, _, err := c.Scheme().ObjectKinds(obj)
	if err != nil {
		return err
	}

	kind := strings.ToLower(gvks[0].Kind)

	// keep the desired state in case it has to be recreated
	desired := obj.DeepCopyObject()

	result, err := retryCreateOrUpdate(ctx, c, obj, f)

	if errors.Is(err, ErrRecreate) {
		logd.FromContext(ctx).Info("recreating "+kind, "reason", err.Error())

		result, err = recreate(ctx, c, obj, desired, f)
	}

	if err != nil {
		return fmt.Errorf("create or update %s: %w", kind, err)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		logd.FromContext(ctx).Info("created " + kind)
	case controllerutil.OperationResultUpdated:
		logd.FromContext(ctx).Info("updated " + kind)
	}

	return nil
}

func retryCreateOrUpdate(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	var result controllerutil.OperationResult

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error

		result, err = controllerutil.CreateOrUpdate(ctx, c, obj, f)

		return err
	})

	return result, err
}

// recreate deletes obj and runs create or update again, starting from the desired state.
func recreate(ctx context.Context, c client.Client, obj client.Object, desired runtime.Object, f controllerutil.MutateFn) (controllerutil.OperationResult, error) {
	if err := c.Delete(ctx, obj); client.IgnoreNotFound(err) != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("delete for recreate: %w", err)
	}

	// Get does not reset obj on NotFound, so the stale state of the deleted object has to be dropped
	reflect.ValueOf(obj).Elem().Set(reflect.ValueOf(desired).Elem())
	obj.SetResourceVersion("")
	obj.SetUID("")

	return retryCreateOrUpdate(ctx, c, obj, f)
}

// ApplyStatus uses SSA to apply the status subresource. Populates missing type meta fields for unit tests.
// Returns an error if the object is not registered with the shared operator scheme.
func ApplyStatus(ctx context.Context, c client.Client, obj client.Object) error {
	// Apply complains if these fields are set
	obj.SetManagedFields(nil)
	// Disable optimistic locking
	obj.SetResourceVersion("")

	// Some fields are not set by the fake client
	if gvk := obj.GetObjectKind().GroupVersionKind(); gvk.Kind == "" {
		kinds, _, err := c.Scheme().ObjectKinds(obj)
		if err != nil {
			// this should actually be a panic because if that's the case no client request will ever succeed.
			return err
		}

		obj.GetObjectKind().SetGroupVersionKind(kinds[0])
	}

	//nolint:staticcheck // client.Apply is deprecated, but our repo does not support generating ApplyConfiguration
	return c.SubResource("status").Patch(ctx, obj, client.Apply, client.FieldOwner("dynatrace-operator"), client.ForceOwnership)
}
