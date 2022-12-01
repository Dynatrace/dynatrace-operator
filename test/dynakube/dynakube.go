package dynakube

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/pkg/errors"
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
	Name      = "dynakube"
	Namespace = "dynatrace"
)

type Builder struct {
	dynakube dynatracev1beta1.DynaKube
}

func NewBuilder() Builder {
	return Builder{
		dynakube: dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{},
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
		Name:        Name,
		Namespace:   Namespace,
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
	dynakubeBuilder.dynakube.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{
		Capabilities: []dynatracev1beta1.CapabilityDisplayName{
			dynatracev1beta1.KubeMonCapability.DisplayName,
			dynatracev1beta1.DynatraceApiCapability.DisplayName,
			dynatracev1beta1.RoutingCapability.DisplayName,
			dynatracev1beta1.MetricsIngestCapability.DisplayName,
			dynatracev1beta1.StatsdIngestCapability.DisplayName,
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
			"inject": "dynakube",
		},
	})
}

func (dynakubeBuilder Builder) Proxy(proxy *dynatracev1beta1.DynaKubeProxy) Builder {
	dynakubeBuilder.dynakube.Spec.Proxy = proxy
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) WithIstio() Builder {
	dynakubeBuilder.dynakube.Spec.EnableIstio = true
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) Privileged() Builder {
	dynakubeBuilder.dynakube.Annotations[dynatracev1beta1.AnnotationFeatureRunOneAgentContainerPrivileged] = "true"
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) ClassicFullstack(classicFullStackSpec *dynatracev1beta1.HostInjectSpec) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.ClassicFullStack = classicFullStackSpec
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) CloudNative(cloudNativeFullStackSpec *dynatracev1beta1.CloudNativeFullStackSpec) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.CloudNativeFullStack = cloudNativeFullStackSpec
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) ApplicationMonitoring(applicationMonitoringSpec *dynatracev1beta1.ApplicationMonitoringSpec) Builder {
	dynakubeBuilder.dynakube.Spec.OneAgent.ApplicationMonitoring = applicationMonitoringSpec
	return dynakubeBuilder
}

func (dynakubeBuilder Builder) Build() dynatracev1beta1.DynaKube {
	return dynakubeBuilder.dynakube
}

func Apply(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta1.AddToScheme(environmentConfig.Client().Resources().GetScheme()))
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &dynakube))

		return ctx
	}
}

func DeleteIfExists(dynakube dynatracev1beta1.DynaKube) func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		resources := environmentConfig.Client().Resources()

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())

		if err != nil {
			return ctx, errors.WithStack(err)
		}

		err = resources.Delete(ctx, &dynakube)
		isNoKindMatchErr := meta.IsNoMatchError(err)

		if err != nil {
			if k8serrors.IsNotFound(err) || isNoKindMatchErr {
				// If the dynakube itself or the crd does not exist, everything is fine
				err = nil
			}

			return ctx, errors.WithStack(err)
		}

		err = wait.For(conditions.New(resources).ResourceDeleted(&dynakube))

		return ctx, err
	}
}

func WaitForDynakubePhase(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		require.NoError(t, wait.For(conditions.New(resources).ResourceMatch(&dynakube, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynatracev1beta1.DynaKube)
			return isDynakube && dynakube.Status.Phase == dynatracev1beta1.Running
		})))

		return ctx
	}
}
