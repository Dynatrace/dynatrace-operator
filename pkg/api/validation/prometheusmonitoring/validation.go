// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
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

type validatorFunc func(ctx context.Context, apiReader client.Reader, pm *prometheusmonitoring.PrometheusMonitoring) string

func (v *Validator) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	ctx, _ = logd.NewFromContext(ctx, "validation")

	pm, err := getPrometheusMonitoring(obj)
	if err != nil {
		return nil, err
	}

	validations := []validatorFunc{
		missingPrometheusCRDs,
		overlappingSelectorWarning,
	}

	var warnings admission.Warnings
	for _, validationTask := range validations {
		if msg := validationTask(ctx, v.apiReader, pm); msg != "" {
			warnings = append(warnings, msg)
		}
	}

	return warnings, nil
}

func getPrometheusMonitoring(obj runtime.Object) (pm *prometheusmonitoring.PrometheusMonitoring, err error) {
	switch v := obj.(type) {
	case *prometheusmonitoring.PrometheusMonitoring:
		pm = v
	default:
		return nil, fmt.Errorf("unknown object %T", obj)
	}

	return
}
