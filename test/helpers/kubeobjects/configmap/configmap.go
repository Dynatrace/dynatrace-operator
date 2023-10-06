//go:build e2e

package configmap

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"testing"

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
	configMap corev1.ConfigMap
}

func NewBuilder(name, namespace string, data map[string]string) Builder {
	return Builder{
		configMap: corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: data,
		},
	}
}

func (b Builder) Build() corev1.ConfigMap {
	return b.configMap
}

func Delete(name string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		configMap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}

		err := envConfig.Client().Resources().Delete(ctx, &configMap, func(options *metav1.DeleteOptions) {
			options.GracePeriodSeconds = address.Of[int64](0)
		})

		if err != nil {
			if k8serrors.IsNotFound(err) {
				err = nil
			}
			require.NoError(t, err)
			return ctx
		}

		resources := envConfig.Client().Resources()
		err = wait.For(conditions.New(resources).ResourceDeleted(&configMap))
		require.NoError(t, err)

		return ctx
	}
}

func Create(configMap corev1.ConfigMap) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Create(ctx, &configMap)
		if err != nil {
			if k8serrors.IsAlreadyExists(err) {
				err = nil
			}
			require.NoError(t, err)
		}

		return ctx
	}
}
