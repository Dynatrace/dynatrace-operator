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
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type DynaKubeType interface {
	*v1beta3.DynaKube | *v1beta4.DynaKube | *v1beta5.DynaKube | *dynakube.DynaKube
}

type validator[T DynaKubeType] struct {
	*validatorClient
}

type validatorClient struct {
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
		noMappedHostPaths,
		extensionsWithoutK8SMonitoring,
		hostPathDatabaseVolumeFound,
		disabledMetadataEnrichmentForInjectionModes,
	}
	updateValidatorErrorFuncs = []updateValidatorFunc{
		IsMutatedAPIURL,
	}
)

type validatorFunc func(ctx context.Context, dv *validatorClient, dk *dynakube.DynaKube) string
type updateValidatorFunc func(ctx context.Context, dv *validatorClient, oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube) string

func newGenericValidator[T DynaKubeType](vi *validatorClient) *validator[T] {
	return &validator[T]{
		validatorClient: vi,
	}
}

func newClient(apiReader client.Reader, cfg *rest.Config) *validatorClient {
	return &validatorClient{
		apiReader: apiReader,
		cfg:       cfg,
		modules:   installconfig.GetModules(),
	}
}

func (v *validator[T]) ValidateCreate(ctx context.Context, obj T) (warnings admission.Warnings, err error) {
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

func (v *validator[T]) ValidateUpdate(ctx context.Context, oldObj, newObj T) (warnings admission.Warnings, err error) {
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

func (v *validator[T]) ValidateDelete(ctx context.Context, obj T) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (v *validator[T]) runValidators(ctx context.Context, validators []validatorFunc, dk *dynakube.DynaKube) []string {
	results := []string{}

	for _, validate := range validators {
		if errMsg := validate(ctx, v.validatorClient, dk); errMsg != "" {
			results = append(results, errMsg)
		}
	}

	return results
}

func (v *validator[T]) runUpdateValidators(ctx context.Context, updateValidators []updateValidatorFunc, oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube) []string {
	results := []string{}

	for _, validate := range updateValidators {
		if errMsg := validate(ctx, v.validatorClient, oldDk, newDk); errMsg != "" {
			results = append(results, errMsg)
		}
	}

	return results
}

func getDynakube[T DynaKubeType](obj T) (dk *dynakube.DynaKube, err error) {
	dk = &dynakube.DynaKube{}

	switch v := any(obj).(type) {
	case *dynakube.DynaKube:
		dk = v
	case *v1beta5.DynaKube:
		err = v.ConvertTo(dk)
	case *v1beta4.DynaKube:
		err = v.ConvertTo(dk)
	case *v1beta3.DynaKube:
		err = v.ConvertTo(dk)
	default:
		return nil, fmt.Errorf("unknown object %T", obj)
	}

	return
}
