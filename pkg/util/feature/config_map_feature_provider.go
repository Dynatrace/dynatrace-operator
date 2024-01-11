package feature

import (
	"context"

	"github.com/open-feature/go-sdk/openfeature"
	"google.golang.org/appengine/log"
)

func NewConfigMapFeatureProvider( /* configure */ ) *ConfigMapFeatureProvider {
	/* configure */
	return &ConfigMapFeatureProvider{}
}

// ConfigMapFeatureProvider implements the OpenFeatureProvider interface
type ConfigMapFeatureProvider struct {
	// cach the file contents?

}

var _ openfeature.FeatureProvider = ConfigMapFeatureProvider{}

func (k ConfigMapFeatureProvider) Metadata() openfeature.Metadata {
	// return information about supported OneAgent versions, release date, OneAgent ConfigMap version
	return openfeature.Metadata{
		Name: "OneAgentVersionConfig",
	}
}

// REQUIRED functionality: boolean, int, string and object
// imho nice basic provider can be found at https://github.com/open-feature/go-sdk-contrib/blob/main/providers/from-env/pkg/provider.go
// needs to be adapted to ConfigMaps
func (k ConfigMapFeatureProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	// perhaps we could use the context to do the mapping of OneAgent versions to specific flag values?
	oneAgentVersion := evalCtx[openfeature.TargetingKey].(string)
	if !OneAgentVersionSupported(oneAgentVersion) {
		log.Warningf(ctx, "Oneagent Version %s not supported", oneAgentVersion)
		return openfeature.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: generateOneAgentVersionNotSupportedResultionDetail(),
		}
	}
	res := k.resolveFlag(flag, defaultValue, evalCtx)
	v, ok := res.Value.(bool)
	if !ok {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(""),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	return openfeature.BoolResolutionDetail{
		Value:                    v,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func OneAgentVersionSupported(version string) bool {
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

// REQUIRED functionality: boolean, int, string and object
func (k ConfigMapFeatureProvider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {

	return openfeature.StringResolutionDetail{}
}

// not 100% sure on that, as numbers are required for providers while float should be supported in the client,
// but as I do not see much benefit in not implementing it just do it
func (k ConfigMapFeatureProvider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	// TODO- see boolean impl or https://github.com/open-feature/go-sdk-contrib/blob/main/providers/from-env/pkg/provider.go for inspiration
	return openfeature.FloatResolutionDetail{}
}

// REQUIRED functionality: boolean, int, string and object
func (k ConfigMapFeatureProvider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	// TODO- see boolean impl or https://github.com/open-feature/go-sdk-contrib/blob/main/providers/from-env/pkg/provider.go for inspiration
	return openfeature.IntResolutionDetail{}
}

// REQUIRED functionality: boolean, int, string and object
func (k ConfigMapFeatureProvider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	// TODO- see boolean impl or https://github.com/open-feature/go-sdk-contrib/blob/main/providers/from-env/pkg/provider.go for inspiration
	return openfeature.InterfaceResolutionDetail{}
}

func (k ConfigMapFeatureProvider) Hooks() []openfeature.Hook {
	// OPTIONAL: while the method has to be there, according to spec a hook mechanism implementation is not mandatory
	// why we might want to implement it: logging and tracing...
	return []openfeature.Hook{}
}

func (k ConfigMapFeatureProvider) resolveFlag(flag string, value bool, ctx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	// do the mapping to the correct versions values here.
	return openfeature.InterfaceResolutionDetail{}
}
