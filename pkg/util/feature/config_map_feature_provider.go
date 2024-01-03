package feature

import (
	"context"
	"fmt"
	"github.com/open-feature/go-sdk/openfeature"
	"google.golang.org/appengine/log"
	"strconv"
)

// ConfigMapFeatureProvider implements the OpenFeatureProvider interface
type ConfigMapFeatureProvider struct {
	// cach the file contents?

}

var _ openfeature.FeatureProvider = &ConfigMapFeatureProvider{}

func (k *ConfigMapFeatureProvider) Metadata() openfeature.Metadata {
	// return information about supported OneAgent versions?
	panic("implement me")
}

func (k *ConfigMapFeatureProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	oneagentVersion := evalCtx[openfeature.TargetingKey].(string)
	if !oneagentVersionSupported(oneagentVersion) {
		log.Warningf(ctx, "Oneagent Version %s not supported", oneagentVersion)
		return openfeature.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: generateOneAgentVersionNotSupportedResultionDetail(),
		}
	}
	err, stringValue := getFlag(ctx, oneagentVersion, flag)
	if err != nil {
		// handle error
		return openfeature.BoolResolutionDetail{}
	}
	// return happy path
	return createBoolResolutionDetail(stringValue)
}

func createBoolResolutionDetail(value string) openfeature.BoolResolutionDetail {
	boolValue, err := strconv.ParseBool(value)

	if err != nil {
		fmt.Println("Error:", err)
		return generateConversionErrorResolutionDetail(value)
	}
	return openfeature.BoolResolutionDetail{
		Value:                    boolValue,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{},
	}
}

func generateConversionErrorResolutionDetail(value string) openfeature.BoolResolutionDetail {
	return openfeature.BoolResolutionDetail{} // fill in error msg
}

func getFlag(ctx context.Context, version string, flag string) (error, string) {
	// load file for oneagnetVersion
	// looks up flag
	// return stringValue if found; err otherwise
	return nil, ""
}

func oneagentVersionSupported(version string) bool {
	// iterate over all .property files mounted in the /supported-oneagent-versions dir
	return true
}

func generateOneAgentVersionNotSupportedResultionDetail() openfeature.ProviderResolutionDetail {
	return openfeature.ProviderResolutionDetail{
		ResolutionError: openfeature.ResolutionError{},
		Reason:          "OneAgent version not supported",
		Variant:         "",
		FlagMetadata:    nil,
	}
}

func (k *ConfigMapFeatureProvider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	// TODO implement me
	panic("implement me")
}

func (k *ConfigMapFeatureProvider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	// TODO implement me
	panic("implement me")
}

func (k *ConfigMapFeatureProvider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	// TODO implement me
	panic("implement me")
}

func (k *ConfigMapFeatureProvider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	// TODO implement me
	panic("implement me")
}

func (k *ConfigMapFeatureProvider) Hooks() []openfeature.Hook {
	// TODO implement me
	panic("implement me")
}
