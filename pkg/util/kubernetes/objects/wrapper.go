// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sobject

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ErrRecreate signal that an immutable field of the existing object has to change, so the object must be deleted and created again.
var ErrRecreate = errors.New("object has to be recreated")

// RetryCreateOrUpdate is a thin wrapper around [controllerutil.CreateOrUpdate] that retries on conflict errors to prevent redoing a whole reconcile.
// If the mutate function returns [ErrRecreate], the existing object is deleted and created again on the next retry.
// Prefer this for external resources, i.e. ones that are not managed by the Dynatrace Operator.
func RetryCreateOrUpdate(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) error {
	gvks, _, err := c.Scheme().ObjectKinds(obj)
	if err != nil {
		return err
	}

	kind := strings.ToLower(gvks[0].Kind)
	group := strings.ToLower(gvks[0].Group)

	var result controllerutil.OperationResult

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var fnerr error

		result, fnerr = controllerutil.CreateOrUpdate(ctx, c, obj, f)
		if !errors.Is(fnerr, ErrRecreate) {
			return fnerr
		}

		logd.FromContext(ctx).Info("recreating "+kind, "reason", fnerr.Error())

		if err := c.Delete(ctx, obj); client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("delete for recreate: %w", err)
		}

		// Wait for deletion, the client, on creation, may read from a cache that still contains the deleted object
		if err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 5*time.Second, true, func(ctx context.Context) (bool, error) {
			geterr := c.Get(ctx, client.ObjectKeyFromObject(obj), obj.DeepCopyObject().(client.Object))
			if k8serrors.IsNotFound(geterr) {
				return true, nil
			}

			return false, geterr
		}); err != nil {
			return fmt.Errorf("wait for deletion: %w", err)
		}

		// Get does not reset obj on NotFound, but Create rejects a set resource version and keeps the managed fields
		obj.SetResourceVersion("")
		obj.SetManagedFields(nil)

		return k8serrors.NewConflict(schema.GroupResource{Group: group, Resource: kind}, obj.GetName(), fnerr)
	})
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
