//go:build e2e

package configmap

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

func New(name, namespace string, data map[string]string) corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func Create(configMap corev1.ConfigMap) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Create(ctx, &configMap)
		if err != nil {
			if k8serrors.IsAlreadyExists(err) {
				err = envConfig.Client().Resources().Update(ctx, &configMap)
			}
			require.NoError(t, err)
		}

		return ctx
	}
}
