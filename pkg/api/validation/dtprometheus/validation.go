// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Validator struct {
	apiReader client.Reader
}

func New(apiReader client.Reader) admission.Validator[runtime.Object] {
	return &Validator{apiReader: apiReader}
}

func (v *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *Validator) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *Validator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *Validator) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	ctx, _ = logd.NewFromContext(ctx, "validation")

	if _, err := getDTPrometheus(obj); err != nil {
		return nil, err
	}

	var warnings admission.Warnings
	if msg := missingPrometheusCRDs(ctx, v.apiReader); msg != "" {
		warnings = append(warnings, msg)
	}

	return warnings, nil
}

func getDTPrometheus(obj runtime.Object) (*dtprometheus.DTPrometheus, error) {
	if dtp, ok := obj.(*dtprometheus.DTPrometheus); ok {
		return dtp, nil
	}

	if gvk := obj.GetObjectKind().GroupVersionKind(); !gvk.Empty() {
		return nil, fmt.Errorf("unknown object %s", gvk)
	}

	return nil, fmt.Errorf("unknown object %T", obj)
}
