// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sobjects

import (
	"context"

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
