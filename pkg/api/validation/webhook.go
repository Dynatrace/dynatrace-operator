// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// typedValidator adapts an admission.Validator[runtime.Object] to admission.Validator[T]
// so that controller-runtime's generic WithValidator can resolve T's concrete reflect.Type.
type typedValidator[T runtime.Object] struct {
	validator admission.Validator[runtime.Object]
}

func (t typedValidator[T]) ValidateCreate(ctx context.Context, obj T) (admission.Warnings, error) {
	return t.validator.ValidateCreate(ctx, obj)
}

func (t typedValidator[T]) ValidateUpdate(ctx context.Context, oldObj, newObj T) (admission.Warnings, error) {
	return t.validator.ValidateUpdate(ctx, oldObj, newObj)
}

func (t typedValidator[T]) ValidateDelete(ctx context.Context, obj T) (admission.Warnings, error) {
	return t.validator.ValidateDelete(ctx, obj)
}

// SetupWebhookForType registers a validating webhook for the concrete API type T using a
// validator that operates on runtime.Object. T must be passed as a concrete pointer type
// (e.g. &dynakube.DynaKube{}) so controller-runtime can resolve its reflect.Type.
func SetupWebhookForType[T runtime.Object](mgr ctrl.Manager, obj T, validator admission.Validator[runtime.Object]) error {
	return ctrl.NewWebhookManagedBy(mgr, obj).WithValidator(typedValidator[T]{validator: validator}).Complete()
}
