//go:build e2e

package secret

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func New(name, namespace string, data map[string][]byte) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func Create(secret corev1.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Create(ctx, &secret)
		if err != nil {
			if k8serrors.IsAlreadyExists(err) {
				err = envConfig.Client().Resources().Update(ctx, &secret)
			}
			require.NoError(t, err)
		}

		return ctx
	}
}

func Delete(secret corev1.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Delete(ctx, &secret)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				err = nil
			}
		}
		require.NoError(t, err)

		return ctx
	}
}
