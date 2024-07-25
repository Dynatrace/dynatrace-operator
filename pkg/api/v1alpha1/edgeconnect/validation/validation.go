package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/validation"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Validator struct {
	apiReader client.Reader
	cfg       *rest.Config
}

type validatorFunc func(ctx context.Context, dv *Validator, ec *edgeconnect.EdgeConnect) string

var validatorErrorFuncs = []validatorFunc{
	isInvalidApiServer,
	nameTooLong,
	checkHostPatternsValue,
	isInvalidServiceName,
	automationRequiresProvisionerValidation,
}

func New(apiReader client.Reader, cfg *rest.Config) admission.CustomValidator {
	return &Validator{
		apiReader: apiReader,
		cfg:       cfg,
	}
}

func (v *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (_ admission.Warnings, err error) {
	ec := obj.(*edgeconnect.EdgeConnect)
	validationErrors := v.runValidators(ctx, validatorErrorFuncs, ec)

	if len(validationErrors) > 0 {
		err = errors.New(validation.SumErrors(validationErrors, "EdgeConnect"))
	}

	return
}

func (v *Validator) ValidateUpdate(ctx context.Context, _, newObj runtime.Object) (warnings admission.Warnings, err error) {
	ec := newObj.(*edgeconnect.EdgeConnect)
	validationErrors := v.runValidators(ctx, validatorErrorFuncs, ec)

	if len(validationErrors) > 0 {
		err = errors.New(validation.SumErrors(validationErrors, "EdgeConnect"))
	}

	return
}

func (v *Validator) ValidateDelete(_ context.Context, _ runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (v *Validator) runValidators(ctx context.Context, validators []validatorFunc, ec *edgeconnect.EdgeConnect) []string {
	results := []string{}

	for _, validate := range validators {
		if errMsg := validate(ctx, v, ec); errMsg != "" {
			results = append(results, errMsg)
		}
	}

	return results
}
