//go:build e2e

package dynakube

import (
	"context"
	"strconv"
	"strings"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	defaultName      = "dynakube"
	defaultNamespace = "dynatrace"
)

type Builder struct {
	dynakube dynatracev1.DynaKube
}

func NewBuilder() Builder {
	return Builder{
		dynakube: dynatracev1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
			Spec: dynatracev1.DynaKubeSpec{},
		},
	}
}

func (dynakubeBuilder Builder) Name(name string) Builder {
	dynakubeBuilder.dynakube.Name = name
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) Namespace(namespace string) Builder {
	dynakubeBuilder.dynakube.Namespace = namespace
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) WithDefaultObjectMeta() Builder {
	dynakubeBuilder.dynakube.ObjectMeta = metav1.ObjectMeta{
		Name:        defaultName,
		Namespace:   defaultNamespace,
		Annotations: map[string]string{},
	}

	return dynakubeBuilder
}

func (dynakubeBuilder Builder) WithAnnotations(annotations map[string]string) Builder {
	for key, value := range annotations {
		dynakubeBuilder.dynakube.ObjectMeta.Annotations[key] = value
	}
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) ApiUrl(apiUrl string) Builder {
	dynakubeBuilder.dynakube.Spec.APIURL = apiUrl
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) WithActiveGate() Builder {
	dynakubeBuilder.dynakube.Spec.ActiveGate = dynatracev1.ActiveGateSpec{
		Capabilities: []dynatracev1.CapabilityDisplayName{
			dynatracev1.KubeMonCapability.DisplayName,
			dynatracev1.DynatraceApiCapability.DisplayName,
			dynatracev1.RoutingCapability.DisplayName,
			dynatracev1.MetricsIngestCapability.DisplayName,
		},
	}
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) Tokens(secretName string) Builder {
	dynakubeBuilder.dynakube.Spec.Tokens = secretName
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) NamespaceSelector(selector metav1.LabelSelector) Builder {
	dynakubeBuilder.dynakube.Spec.NamespaceSelector = selector
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) WithDynakubeNamespaceSelector() Builder {
	return dynakubeBuilder.NamespaceSelector(metav1.LabelSelector{
		MatchLabels: map[string]string{
			"inject": dynakubeBuilder.dynakube.Name,
		},
	})
}

func (dynakubeBuilder Builder) Proxy(proxy *dynatracev1.DynaKubeProxy) Builder {
	dynakubeBuilder.dynakube.Spec.Proxy = proxy
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) WithIstio() Builder {
	dynakubeBuilder.dynakube.Spec.EnableIstio = true
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) Privileged() Builder {
	dynakubeBuilder.dynakube.Annotations[dynatracev1.AnnotationFeatureRunOneAgentContainerPrivileged] = "true"
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) ClassicFullstack(classicFullStackSpec *dynatracev1.HostInjectSpec) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.ClassicFullStack = classicFullStackSpec
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) CloudNative(cloudNativeFullStackSpec *dynatracev1.CloudNativeFullStackSpec) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.CloudNativeFullStack = cloudNativeFullStackSpec
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) CloudNativeWithAgentVersion(cloudNativeFullStackSpec *dynatracev1.CloudNativeFullStackSpec, version version.SemanticVersion) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.CloudNativeFullStack = cloudNativeFullStackSpec
	dynakubeBuilder.dynakube.Spec.OneAgent.CloudNativeFullStack.Version = version.String()
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) ApplicationMonitoring(applicationMonitoringSpec *dynatracev1.ApplicationMonitoringSpec) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.ApplicationMonitoring = applicationMonitoringSpec
	return dynakubeBuilder
}

func (builder Builder) WithSyntheticLocation(entityId string) Builder {
	builder.dynakube.Annotations[dynatracev1.AnnotationFeatureSyntheticLocationEntityId] = entityId
	return builder
}

func (dynakubeBuilder Builder) Build() dynatracev1.DynaKube {
	return dynakubeBuilder.dynakube
}

func Create(dynakube dynatracev1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1.AddToScheme(environmentConfig.Client().Resources().GetScheme()))
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &dynakube))
		return ctx
	}
}

func Update(dynakube dynatracev1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1.AddToScheme(environmentConfig.Client().Resources().GetScheme()))
		var dk dynatracev1.DynaKube
		require.NoError(t, environmentConfig.Client().Resources().Get(ctx, dynakube.Name, dynakube.Namespace, &dk))
		dynakube.ResourceVersion = dk.ResourceVersion
		require.NoError(t, environmentConfig.Client().Resources().Update(ctx, &dynakube))
		return ctx
	}
}

func Delete(dynakube dynatracev1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		err := dynatracev1.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Delete(ctx, &dynakube)
		isNoKindMatchErr := meta.IsNoMatchError(err)

		if err != nil {
			if k8serrors.IsNotFound(err) || isNoKindMatchErr {
				// If the dynakube itself or the crd does not exist, everything is fine
				err = nil
			}
			require.NoError(t, err)
		}

		err = wait.For(conditions.New(resources).ResourceDeleted(&dynakube))
		require.NoError(t, err)
		return ctx
	}
}

func WaitForDynakubePhase(dynakube dynatracev1.DynaKube, phase dynatracev1.DynaKubePhaseType) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		err := wait.For(conditions.New(resources).ResourceMatch(&dynakube, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynatracev1.DynaKube)
			return isDynakube && dynakube.Status.Phase == phase
		}))

		require.NoError(t, err)

		return ctx
	}
}

func SyntheticLocationOrdinal(dynakube dynatracev1.DynaKube) uint64 {
	const defaultOrd = uint64(0)
	_, suffix, found := strings.Cut(dynakube.FeatureSyntheticLocationEntityId(), "-")
	if !found {
		return defaultOrd
	}

	parsed, err := strconv.ParseUint(suffix, 16, 64)
	if err != nil {
		return defaultOrd
	}

	return parsed
}
