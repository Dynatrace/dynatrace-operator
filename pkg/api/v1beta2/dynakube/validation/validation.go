package validation

import (
	"context"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Validator struct {
	apiReader client.Reader
	cfg       *rest.Config
}

var (
	validatorErrorFuncs = []validatorFunc{
		NoApiUrl,
		IsInvalidApiUrl,
		IsThirdGenAPIUrl,
		missingCSIDaemonSet,
		disabledCSIForReadonlyCSIVolume,
		invalidActiveGateCapabilities,
		duplicateActiveGateCapabilities,
		invalidActiveGateProxyUrl,
		conflictingOneAgentConfiguration,
		conflictingNodeSelector,
		conflictingNamespaceSelector,
		noResourcesAvailable,
		imageFieldSetWithoutCSIFlag,
		conflictingOneAgentVolumeStorageSettings,
		nameViolatesDNS1035,
		nameTooLong,
		namespaceSelectorViolateLabelSpec,
		imageFieldHasTenantImage,
	}
	validatorWarningFuncs = []validatorFunc{
		missingActiveGateMemoryLimit,
		unsupportedOneAgentImage,
		conflictingHostGroupSettings,
		deprecatedFeatureFlag,
	}
)

type validatorFunc func(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string

func New(apiReader client.Reader, cfg *rest.Config) admission.CustomValidator {
	return &Validator{
		apiReader: apiReader,
		cfg:       cfg,
	}
}

func (v *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	dk := obj.(*dynakube.DynaKube)
	errMessages := v.runValidators(ctx, validatorErrorFuncs, dk)
	warnings = v.runValidators(ctx, validatorWarningFuncs, dk)

	if len(errMessages) != 0 {
		err = errors.New(validation.SumErrors(errMessages, "DynaKube"))
	}

	return
}

func (v *Validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	dk := newObj.(*dynakube.DynaKube)
	errMessages := v.runValidators(ctx, validatorErrorFuncs, dk)
	warnings = v.runValidators(ctx, validatorWarningFuncs, dk)

	if len(errMessages) != 0 {
		err = errors.New(validation.SumErrors(errMessages, "DynaKube"))
	}

	return
}

func (v *Validator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (v *Validator) runValidators(ctx context.Context, validators []validatorFunc, dk *dynakube.DynaKube) []string {
	results := []string{}

	for _, validate := range validators {
		if errMsg := validate(ctx, v, dk); errMsg != "" {
			results = append(results, errMsg)
		}
	}

	return results
}
