package namespace

import (
	"context"
	"testing"

	"github.com/pkg/errors"
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

func (namespaceBuilder Builder) Build() corev1.Namespace {
	return namespaceBuilder.namespace
}

func Delete(name string) func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		var namespace corev1.Namespace
		err := environmentConfig.Client().Resources().Get(ctx, name, "", &namespace)

		if err != nil {
			return ctx, errors.WithStack(err)
		}

		err = environmentConfig.Client().Resources().Delete(ctx, &namespace)

		if err != nil {
			return ctx, errors.WithStack(err)
		}

		resources := environmentConfig.Client().Resources()
		err = wait.For(conditions.New(resources).ResourceDeleted(&namespace))

		return ctx, errors.WithStack(err)
	}
}

func DeleteIfExists(name string) func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		err := environmentConfig.Client().Resources().Delete(ctx, &namespace)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				err = nil
			}

			return ctx, errors.WithStack(err)
		}

		resources := environmentConfig.Client().Resources()
		err = wait.For(conditions.New(resources).ResourceDeleted(&namespace))

		return ctx, errors.WithStack(err)
	}
}

func Recreate(namespace corev1.Namespace) func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		ctx, err := Delete(namespace.Name)(ctx, environmentConfig, t)

		if err != nil && !k8serrors.IsNotFound(errors.Cause(err)) {
			return ctx, err
		}

		createNamespace := namespace
		err = environmentConfig.Client().Resources().Create(ctx, &createNamespace)
		return ctx, errors.WithStack(err)
	}
}

func Create(namespace corev1.Namespace) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &namespace))

		return ctx
	}
}
