//go:build e2e

package namespace

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type Builder struct {
	namespace corev1.Namespace
}

func NewBuilder(name string) Builder {
	return Builder{
		namespace: corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}
}

func (namespaceBuilder Builder) WithLabels(labels map[string]string) Builder {
	namespaceBuilder.namespace.ObjectMeta.Labels = labels
	return namespaceBuilder
}

func (namespaceBuilder Builder) WithAnnotation(annotations map[string]string) Builder {
	namespaceBuilder.namespace.ObjectMeta.Annotations = annotations
	return namespaceBuilder
}

func (namespaceBuilder Builder) Build() corev1.Namespace {
	return namespaceBuilder.namespace
}

func Delete(namespaceName string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		err := environmentConfig.Client().Resources().Delete(ctx, &namespace, func(options *metav1.DeleteOptions) {
			options.GracePeriodSeconds = address.Of[int64](0)
		})

		if err != nil {
			if k8serrors.IsNotFound(err) {
				err = nil
			}
			require.NoError(t, err)
			return ctx
		}

		resources := environmentConfig.Client().Resources()
		err = wait.For(conditions.New(resources).ResourceDeleted(&namespace))
		require.NoError(t, err)

		return ctx
	}
}

func Create(namespace corev1.Namespace) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		err := environmentConfig.Client().Resources().Create(ctx, &namespace)
		if err != nil {
			if k8serrors.IsAlreadyExists(err) {
				err = nil
			}
			require.NoError(t, err)
		}

		ctx, _ = istio.AddIstioNetworkAttachment(namespace)(ctx, environmentConfig, t)
		return ctx
	}
}
