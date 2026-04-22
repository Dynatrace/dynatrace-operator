package validation

import (
	"context"
	"errors"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	v1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	v1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/validation"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Validator struct {
	apiReader client.Reader
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
		mutuallyExclusiveActiveGatePVsettings,
		invalidActiveGateProxyURL,
		conflictingOneAgentConfiguration,
		conflictingOneAgentNodeSelector,
		conflictingNamespaceSelector,
		isIstioNotInstalled,
		imageFieldSetWithoutCSIFlag,
		missingCodeModulesImage,
		conflictingOneAgentVolumeStorageSettings,
		nameInvalid,
		namespaceSelectorViolateLabelSpec,
		imageFieldHasTenantImage,
		extensionControllerImage,
		extensionControllerPVCStorageDevice,
		tooManyAGReplicas,
		missingKSPMImage,
		kspmWithoutK8SMonitoring,
		mappedHostPathsWithRootPath,
		relativeMappedHostPaths,
		missingLogMonitoringImage,
		emptyTelemetryIngestProtocolsList,
		unknownTelemetryIngestProtocols,
		invalidTelemetryIngestName,
		forbiddenTelemetryIngestServiceNameSuffix,
		conflictingTelemetryIngestServiceNames,
		missingOtelCollectorImage,
		missingDatabaseExecutorImage,
		conflictingOrInvalidDatabasesVolumeMounts,
		unusedDatabasesVolume,
		invalidGlobalResourceAttributes,
		invalidOneAgentResourceAttributes,
		invalidOTLPResourceAttributes,
	}
	validatorWarningFuncs = []validatorFunc{
		missingActiveGateMemoryLimit,
		unsupportedOneAgentImage,
		conflictingHostGroupSettings,
		conflictingMaxUnavailableAnnotationWithRollingUpdate,
		deprecatedAutoUpdate,
		deprecatedOneAgentVersionField,
		deprecatedFeatureFlag,
		ignoredLogMonitoringTemplate,
		conflictingAPIURLForExtensions,
		noMappedHostPaths,
		extensionsWithoutK8SMonitoring,
		hostPathDatabaseVolumeFound,
		disabledMetadataEnrichmentForInjectionModes,
		activeGateRollingUpdateWithOldK8sVersion,
		globalResourceAttributesExceedsLimit,
		oneAgentResourceAttributesExceedsLimit,
		otlpResourceAttributesExceedsLimit,
		deprecatedPaasToken,
	}
	updateValidatorErrorFuncs = []updateValidatorFunc{
		IsMutatedAPIURL,
	}
)

type validatorFunc func(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string
type updateValidatorFunc func(ctx context.Context, dv *Validator, oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube) string

func New(apiReader client.Reader) admission.Validator[runtime.Object] {
	return &Validator{
		apiReader: apiReader,
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
	oldDK, err := getDynakube(oldObj)
	if err != nil {
		return
	}

	newDK, err := getDynakube(newObj)
	if err != nil {
		return
	}

	errMessages := v.runValidators(ctx, validatorErrorFuncs, newDK)
	warnings = v.runValidators(ctx, validatorWarningFuncs, newDK)

	errMessages = append(errMessages, v.runUpdateValidators(ctx, updateValidatorErrorFuncs, oldDK, newDK)...)

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

func (v *Validator) runUpdateValidators(ctx context.Context, updateValidators []updateValidatorFunc, oldDK *dynakube.DynaKube, newDK *dynakube.DynaKube) []string {
	results := []string{}

	for _, validate := range updateValidators {
		if errMsg := validate(ctx, v, oldDK, newDK); errMsg != "" {
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
	default:
		if gvk := obj.GetObjectKind().GroupVersionKind(); !gvk.Empty() {
			return nil, fmt.Errorf("unknown object %s", gvk)
		}

		return nil, fmt.Errorf("unknown object %T", obj)
	}

	return
}
