package validation

import (
	"context"
	"errors"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	v1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	v1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	v1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/validation"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Validator struct {
	apiReader client.Reader
	cfg       *rest.Config
	modules   installconfig.Modules
}

var (
	validatorErrorFuncs = []validatorFunc{
		isActiveGateModuleDisabled,
		isExtensionsModuleDisabled,
		isLogMonitoringModuleDisabled,
		isKSPMDisabled,
		isOneAgentModuleDisabled,
		isOneAgentVersionValid,
		duplicateOneAgentArguments,
		forbiddenHostIDSourceArgument,
		NoAPIURL,
		IsInvalidAPIURL,
		IsThirdGenAPIUrl,
		invalidActiveGateCapabilities,
		duplicateActiveGateCapabilities,
		mutuallyExclusiveActiveGatePVsettings,
		invalidActiveGateProxyURL,
		conflictingOneAgentConfiguration,
		conflictingOneAgentNodeSelector,
		conflictingNamespaceSelector,
		noResourcesAvailable,
		imageFieldSetWithoutCSIFlag,
		missingCodeModulesImage,
		conflictingOneAgentVolumeStorageSettings,
		nameViolatesDNS1035,
		nameTooLong,
		namespaceSelectorViolateLabelSpec,
		imageFieldHasTenantImage,
		extensionControllerImage,
		extensionControllerPVCStorageDevice,
		tooManyAGReplicas,
		missingKSPMImage,
		mappedHostPathsWithRootPath,
		relativeMappedHostPaths,
		missingLogMonitoringImage,
		emptyTelemetryIngestProtocolsList,
		unknownTelemetryIngestProtocols,
		duplicatedTelemetryIngestProtocols,
		invalidTelemetryIngestName,
		forbiddenTelemetryIngestServiceNameSuffix,
		conflictingTelemetryIngestServiceNames,
		missingOtelCollectorImage,
		missingDatabaseExecutorImage,
		conflictingOrInvalidDatabasesVolumeMounts,
		unusedDatabasesVolume,
	}
	validatorWarningFuncs = []validatorFunc{
		missingActiveGateMemoryLimit,
		unsupportedOneAgentImage,
		conflictingHostGroupSettings,
		deprecatedAutoUpdate,
		deprecatedFeatureFlag,
		ignoredLogMonitoringTemplate,
		conflictingAPIURLForExtensions,
		logMonitoringWithoutK8SMonitoring,
		kspmWithoutK8SMonitoring,
		noMappedHostPaths,
		extensionsWithoutK8SMonitoring,
		hostPathDatabaseVolumeFound,
	}
	updateValidatorErrorFuncs = []updateValidatorFunc{
		IsMutatedAPIURL,
	}
)

type validatorFunc func(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string
type updateValidatorFunc func(ctx context.Context, dv *Validator, oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube) string

func New(apiReader client.Reader, cfg *rest.Config) admission.CustomValidator {
	return &Validator{
		apiReader: apiReader,
		cfg:       cfg,
		modules:   installconfig.GetModules(),
	}
}

func (v *Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	dk, err := getDynakube(obj)
	if err != nil {
		return
	}

	errMessages := v.runValidators(ctx, validatorErrorFuncs, dk)
	warnings = v.runValidators(ctx, validatorWarningFuncs, dk)

	if len(errMessages) != 0 {
		err = errors.New(validation.SumErrors(errMessages, "DynaKube"))
	}

	return
}

func (v *Validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	oldDk, err := getDynakube(oldObj)
	if err != nil {
		return
	}

	newDk, err := getDynakube(newObj)
	if err != nil {
		return
	}

	errMessages := v.runValidators(ctx, validatorErrorFuncs, newDk)
	warnings = v.runValidators(ctx, validatorWarningFuncs, newDk)

	errMessages = append(errMessages, v.runUpdateValidators(ctx, updateValidatorErrorFuncs, oldDk, newDk)...)

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

func (v *Validator) runUpdateValidators(ctx context.Context, updateValidators []updateValidatorFunc, oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube) []string {
	results := []string{}

	for _, validate := range updateValidators {
		if errMsg := validate(ctx, v, oldDk, newDk); errMsg != "" {
			results = append(results, errMsg)
		}
	}

	return results
}

func getDynakube(obj runtime.Object) (dk *dynakube.DynaKube, err error) {
	dk = &dynakube.DynaKube{}

	switch v := obj.(type) {
	case *dynakube.DynaKube:
		dk = v
	case *v1beta5.DynaKube:
		err = v.ConvertTo(dk)
	case *v1beta4.DynaKube:
		err = v.ConvertTo(dk)
	case *v1beta3.DynaKube:
		err = v.ConvertTo(dk)
	default:
		if gvk := obj.GetObjectKind().GroupVersionKind(); !gvk.Empty() {
			return nil, fmt.Errorf("unknown object %s", gvk)
		}

		return nil, fmt.Errorf("unknown object %T", obj)
	}

	return
}
