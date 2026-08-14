// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sobject

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RetryCreateOrUpdate is a thin wrapper around [controllerutil.CreateOrUpdate] that retries on conflict errors to prevent redoing a whole reconcile.
// Prefer this for external resources, i.e. ones that are not managed by the Dynatrace Operator.
func RetryCreateOrUpdate(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) (result controllerutil.OperationResult, err error) {
	_ = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err = controllerutil.CreateOrUpdate(ctx, c, obj, f)

		return err
	})

	return
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
		kinds, _, err := scheme.Scheme.ObjectKinds(obj)
		if err != nil {
			// this should actually be a panic because if that's the case no client request will ever succeed.
			return err
		}

		obj.GetObjectKind().SetGroupVersionKind(kinds[0])
	}

	//nolint:staticcheck // client.Apply is deprecated, but our repo does not support generating ApplyConfiguration
	return c.SubResource("status").Patch(ctx, obj, client.Apply, client.FieldOwner("dynatrace-operator"), client.ForceOwnership)
}
