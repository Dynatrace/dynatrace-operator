package namespace

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func Create(name string, labels map[string]string) env.Func {
	return func(ctx context.Context, environmentConfig *envconf.Config) (context.Context, error) {
		namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels}}
		err := environmentConfig.Client().Resources().Create(ctx, namespace)
		return ctx, errors.WithStack(err)
	}
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

func Recreate(name string, labels map[string]string) func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		ctx, err := Delete(name)(ctx, environmentConfig, t)

		if err != nil && !k8serrors.IsNotFound(errors.Cause(err)) {
			return ctx, err
		}

		return Create(name, labels)(ctx, environmentConfig)
	}
}
