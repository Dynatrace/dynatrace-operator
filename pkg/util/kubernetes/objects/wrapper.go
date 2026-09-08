// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sobject

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RetryCreateOrUpdate is a thin wrapper around [controllerutil.CreateOrUpdate] that retries on conflict errors to prevent redoing a whole reconcile.
// Prefer this for external resources, i.e. ones that are not managed by the Dynatrace Operator.
func RetryCreateOrUpdate(ctx context.Context, c client.Client, obj client.Object, f controllerutil.MutateFn) error {
	gvks, _, err := c.Scheme().ObjectKinds(obj)
	if err != nil {
		return err
	}

	kind := strings.ToLower(gvks[0].Kind)

	var result controllerutil.OperationResult

	_ = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err = controllerutil.CreateOrUpdate(ctx, c, obj, f)

		return err
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
