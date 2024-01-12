package feature

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-feature/go-sdk/openfeature"
)

const (
	ErrorFlagNotFound = "flag not found"

	OneAgentVersionMappingKey = "OneAgentVersionMappingKey"
)

// OpenFeature research relevant
func ReadConfigMapAndCreateFeatureProvider(ctx context.Context, apiReader client.Reader) (*ConfigMapFeatureProvider, error) {
	configMap, err := ReadConfigMap(ctx, apiReader)
	if err != nil {
		return nil, err
	}
	return NewConfigMapFeatureProvider(configMap), nil
}

func NewConfigMapFeatureProvider(configMap *corev1.ConfigMap) *ConfigMapFeatureProvider {
	return &ConfigMapFeatureProvider{
		configMap: configMap,
	}
}

func ReadConfigMap(ctx context.Context, apiReader client.Reader) (*corev1.ConfigMap, error) {
	var configMap corev1.ConfigMap
	key := client.ObjectKey{Name: OneAgentVersionMappingKey, Namespace: "dynatrace"} // optimise key generation?
	err := apiReader.Get(ctx, key, &configMap)
	if err != nil {
		return nil, err
	}
	return &configMap, nil
}

// ConfigMapFeatureProvider implements the OpenFeatureProvider interface
type ConfigMapFeatureProvider struct {
	configMap *corev1.ConfigMap
}

var _ openfeature.FeatureProvider = ConfigMapFeatureProvider{}

// REQUIRED as per spec
func (k ConfigMapFeatureProvider) Metadata() openfeature.Metadata {
	// return information about supported OneAgent versions, release date, OneAgent ConfigMap version
	return openfeature.Metadata{
		Name: "OneAgentVersionConfig",
	}
}

// REQUIRED functionality: boolean, int, string and object
// imho nice basic provider can be found at https://github.com/open-feature/go-sdk-contrib/blob/main/providers/from-env/pkg/provider.go
func (k ConfigMapFeatureProvider) BooleanEvaluation(ctx context.Context, flagKey string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	res := k.resolveFlag(flagKey, defaultValue, evalCtx)
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

// REQUIRED functionality: boolean, int, string and object
func (k ConfigMapFeatureProvider) StringEvaluation(ctx context.Context, flagKey string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	res := k.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(string)
	if !ok {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(""),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	return openfeature.StringResolutionDetail{
		Value:                    v,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

// not 100% sure on that, as numbers are required for providers while float should be supported in the client,
// but as I do not see much benefit in not implementing it just do it
func (k ConfigMapFeatureProvider) FloatEvaluation(ctx context.Context, flagKey string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	res := k.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(float64)
	if !ok {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(""),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	return openfeature.FloatResolutionDetail{
		Value:                    v,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

// REQUIRED functionality: boolean, int, string and object
func (k ConfigMapFeatureProvider) IntEvaluation(ctx context.Context, flagKey string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	res := k.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(float64)
	if !ok {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(""),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	return openfeature.IntResolutionDetail{
		Value:                    int64(v),
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

// REQUIRED functionality: boolean, int, string and object
func (k ConfigMapFeatureProvider) ObjectEvaluation(ctx context.Context, flagKey string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return k.resolveFlag(flagKey, defaultValue, evalCtx)
}

// REQUIRED as per spec, but not the actual implementation of the hooking mechanism as I interpret it...
// why we might want to implement it: logging and tracing...
func (k ConfigMapFeatureProvider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

func (p *ConfigMapFeatureProvider) resolveFlag(flagKey string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	// fetch the stored flag from environment variables
	value, found := p.configMap.Data[flagKey] // TODO: insert kube apicall here p.envFetch.fetchStoredFlag(flagKey)
	if !found {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(ErrorFlagNotFound),
				Reason:          openfeature.ErrorReason,
			},
		}
	}
	// ignore evalContext or variants here- look for detail in the imple of
	// https://github.com/open-feature/go-sdk-contrib/blob/main/providers/from-env/pkg/provider.go

	return openfeature.InterfaceResolutionDetail{
		Value: value,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Variant: "ignored for research",
			Reason:  "ignored for research",
		},
	}
}
