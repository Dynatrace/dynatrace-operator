// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"errors"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/validation"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Validator struct {
	apiReader client.Reader
}

type validatorFunc func(ctx context.Context, dv *Validator, dtp *dtprometheus.DTPrometheus) string

var (
	// validatorErrorFuncs is intentionally empty for now: it is the scaffold for
	// future blocking validators, mirroring the dynakube validation package.
	validatorErrorFuncs = []validatorFunc{}

	validatorWarningFuncs = []validatorFunc{
		missingPrometheusCRDs,
	}
)

func New(apiReader client.Reader) admission.Validator[runtime.Object] {
	return &Validator{
		apiReader: apiReader,
	}
}

func (v *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	ctx, _ = logd.NewFromContext(ctx, "validation")

	dtp, err := getDTPrometheus(obj)
	if err != nil {
		return
	}

	errMessages := v.runValidators(ctx, validatorErrorFuncs, dtp)
	warnings = v.runValidators(ctx, validatorWarningFuncs, dtp)

	if len(errMessages) != 0 {
		err = errors.New(validation.SumErrors(errMessages, "DTPrometheus"))
	}

	return
}

func (v *Validator) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	ctx, _ = logd.NewFromContext(ctx, "validation")

	dtp, err := getDTPrometheus(newObj)
	if err != nil {
		return
	}

	errMessages := v.runValidators(ctx, validatorErrorFuncs, dtp)
	warnings = v.runValidators(ctx, validatorWarningFuncs, dtp)

	if len(errMessages) != 0 {
		err = errors.New(validation.SumErrors(errMessages, "DTPrometheus"))
	}

	return
}

func (v *Validator) ValidateDelete(_ context.Context, _ runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (v *Validator) runValidators(ctx context.Context, validators []validatorFunc, dtp *dtprometheus.DTPrometheus) []string {
	results := []string{}

	for _, validate := range validators {
		if msg := validate(ctx, v, dtp); msg != "" {
			results = append(results, msg)
		}
	}

	return results
}

func getDTPrometheus(obj runtime.Object) (dtp *dtprometheus.DTPrometheus, err error) {
	switch v := obj.(type) {
	case *dtprometheus.DTPrometheus:
		dtp = v
	default:
		if gvk := obj.GetObjectKind().GroupVersionKind(); !gvk.Empty() {
			return nil, fmt.Errorf("unknown object %s", gvk)
		}

		return nil, fmt.Errorf("unknown object %T", obj)
	}

	return
}
