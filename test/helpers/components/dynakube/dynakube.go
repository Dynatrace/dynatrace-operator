//go:build e2e

package dynakube

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	v1betadynakube "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
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
	dynakube v1betadynakube.DynaKube
}

func NewBuilder() Builder {
	return Builder{
		dynakube: v1betadynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
			Spec: v1betadynakube.DynaKubeSpec{},
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

func (dynakubeBuilder Builder) WithCustomPullSecret(secretName string) Builder {
	dynakubeBuilder.dynakube.Spec.CustomPullSecret = secretName
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
	dynakubeBuilder.dynakube.Spec.ActiveGate = v1betadynakube.ActiveGateSpec{
		Capabilities: []v1betadynakube.CapabilityDisplayName{
			v1betadynakube.KubeMonCapability.DisplayName,
			v1betadynakube.DynatraceApiCapability.DisplayName,
			v1betadynakube.RoutingCapability.DisplayName,
			v1betadynakube.MetricsIngestCapability.DisplayName,
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

func (dynakubeBuilder Builder) Proxy(proxy *v1betadynakube.DynaKubeProxy) Builder {
	dynakubeBuilder.dynakube.Spec.Proxy = proxy
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) WithIstio() Builder {
	dynakubeBuilder.dynakube.Spec.EnableIstio = true
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) Privileged() Builder {
	dynakubeBuilder.dynakube.Annotations[v1betadynakube.AnnotationFeatureRunOneAgentContainerPrivileged] = "true"
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) ClassicFullstack(classicFullStackSpec *v1betadynakube.HostInjectSpec) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.ClassicFullStack = classicFullStackSpec
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) CloudNative(cloudNativeFullStackSpec *v1betadynakube.CloudNativeFullStackSpec) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.CloudNativeFullStack = cloudNativeFullStackSpec
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) CloudNativeWithAgentVersion(cloudNativeFullStackSpec *v1betadynakube.CloudNativeFullStackSpec, version version.SemanticVersion) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.CloudNativeFullStack = cloudNativeFullStackSpec
	dynakubeBuilder.dynakube.Spec.OneAgent.CloudNativeFullStack.Version = version.String()
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) ApplicationMonitoring(applicationMonitoringSpec *v1betadynakube.ApplicationMonitoringSpec) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.ApplicationMonitoring = applicationMonitoringSpec
	return dynakubeBuilder
}

func (builder Builder) WithSyntheticLocation(entityId string) Builder {
	builder.dynakube.Annotations[v1betadynakube.AnnotationFeatureSyntheticLocationEntityId] = entityId
	return builder
}

func (builder Builder) ResetOneAgent() Builder {
	builder.dynakube.Spec.OneAgent.ClassicFullStack = nil
	builder.dynakube.Spec.OneAgent.CloudNativeFullStack = nil
	builder.dynakube.Spec.OneAgent.ApplicationMonitoring = nil
	builder.dynakube.Spec.OneAgent.HostMonitoring = nil
	return builder
}

func (dynakubeBuilder Builder) Build() v1betadynakube.DynaKube {
	return dynakubeBuilder.dynakube
}

func Create(dynakube v1betadynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, v1beta1.AddToScheme(envConfig.Client().Resources().GetScheme()))
		require.NoError(t, envConfig.Client().Resources().Create(ctx, &dynakube))
		return ctx
	}
}

func Update(dynakube v1betadynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, v1beta1.AddToScheme(envConfig.Client().Resources().GetScheme()))
		var dk v1betadynakube.DynaKube
		require.NoError(t, envConfig.Client().Resources().Get(ctx, dynakube.Name, dynakube.Namespace, &dk))
		dynakube.ResourceVersion = dk.ResourceVersion
		require.NoError(t, envConfig.Client().Resources().Update(ctx, &dynakube))
		return ctx
	}
}

func Delete(dynakube v1betadynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := v1beta1.AddToScheme(resources.GetScheme())
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

func WaitForDynakubePhase(dynakube v1betadynakube.DynaKube, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := wait.For(conditions.New(resources).ResourceMatch(&dynakube, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*v1betadynakube.DynaKube)
			return isDynakube && dynakube.Status.Phase == phase
		}))

		require.NoError(t, err)

		return ctx
	}
}

func SyntheticLocationOrdinal(dynakube v1betadynakube.DynaKube) uint64 {
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
