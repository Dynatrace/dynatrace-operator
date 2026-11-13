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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	gvk := gvks[0]
	kind := strings.ToLower(gvk.Kind)
	group := strings.ToLower(gvk.Group)

	var result controllerutil.OperationResult

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var fnerr error

		result, fnerr = controllerutil.CreateOrUpdate(ctx, c, obj, f)
		if !errors.Is(fnerr, ErrRecreate) {
			return fnerr
		}

		logd.FromContext(ctx).Info("recreating "+kind, "reason", fnerr.Error())

		if delerr := c.Delete(ctx, obj); client.IgnoreNotFound(delerr) != nil {
			return fmt.Errorf("delete for recreate: %w", delerr)
		}

		// Wait for deletion, the client, on creation, may read from a cache that still contains the deleted object
		if waiterr := waitForDeletion(ctx, c, gvk, client.ObjectKeyFromObject(obj)); waiterr != nil {
			return fmt.Errorf("wait for deletion: %w", waiterr)
		}

		// Create rejects a set resource version. Without clearing it the retry fails and the whole reconcile has to run again,
		// which would also fix the recreate, but it is better to handle it in one go.
		obj.SetResourceVersion("")

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

func waitForDeletion(ctx context.Context, c client.Client, gvk schema.GroupVersionKind, key client.ObjectKey) error {
	return wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		meta := &metav1.PartialObjectMetadata{}
		meta.SetGroupVersionKind(gvk)

		err := c.Get(ctx, key, meta)
		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
}
