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

func NewDynakube() dynatracev1beta1.DynaKube {
	return dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: Namespace,
		},
	}
}

func newWithActiveGate(apiUrl string) dynatracev1beta1.DynaKube {
	instance := NewDynakube()
	instance.Spec = dynatracev1beta1.DynaKubeSpec{
		APIURL: apiUrl,
		NamespaceSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"inject": "dynakube",
			},
		},
		ActiveGate: dynatracev1beta1.ActiveGateSpec{
			Capabilities: []dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.KubeMonCapability.DisplayName,
				dynatracev1beta1.DynatraceApiCapability.DisplayName,
				dynatracev1beta1.RoutingCapability.DisplayName,
				dynatracev1beta1.MetricsIngestCapability.DisplayName,
				dynatracev1beta1.StatsdIngestCapability.DisplayName,
			},
		},
	}

	return instance
}

func ApplyClassicFullStack(apiUrl string, classicFullStackSpec *dynatracev1beta1.HostInjectSpec) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta1.AddToScheme(environmentConfig.Client().Resources().GetScheme()))

		instance := newWithActiveGate(apiUrl)
		instance.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureRunOneAgentContainerPrivileged: "true",
		}
		instance.Spec.OneAgent.ClassicFullStack = classicFullStackSpec

		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &instance))

		return ctx
	}
}

func ApplyCloudNative(apiUrl string, cloudNativeFullStackSpec *dynatracev1beta1.CloudNativeFullStackSpec) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta1.AddToScheme(environmentConfig.Client().Resources().GetScheme()))

		instance := newWithActiveGate(apiUrl)
		instance.Spec.OneAgent.CloudNativeFullStack = cloudNativeFullStackSpec

		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &instance))

		return ctx
	}
}

func DeleteIfExists() func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		instance := NewDynakube()
		resources := environmentConfig.Client().Resources()

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())

		if err != nil {
			return ctx, errors.WithStack(err)
		}

		err = resources.Delete(ctx, &instance)
		_, isNoKindMatchErr := err.(*meta.NoKindMatchError)

		if err != nil {
			if k8serrors.IsNotFound(err) || isNoKindMatchErr {
				// If the dynakube itself or the crd does not exist, everything is fine
				err = nil
			}

			return ctx, errors.WithStack(err)
		}

		err = wait.For(conditions.New(resources).ResourceDeleted(&instance))

		return ctx, err
	}
}

func WaitForDynakubePhase() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		instance := NewDynakube()
		resources := environmentConfig.Client().Resources()

		require.NoError(t, wait.For(conditions.New(resources).ResourceMatch(&instance, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynatracev1beta1.DynaKube)
			return isDynakube && dynakube.Status.Phase == dynatracev1beta1.Running
		})))

		return ctx
	}
}
