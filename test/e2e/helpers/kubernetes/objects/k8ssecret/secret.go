// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package k8ssecret

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
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

func WaitForAbsence(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		err := wait.For(conditions.New(resources).ResourceDeleted(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}), wait.WithTimeout(2*time.Minute))
		require.NoError(t, err)

		return ctx
	}
}

func Exists(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Get(ctx, name, namespace, &corev1.Secret{})
		require.NoError(t, err)

		return ctx
	}
}
