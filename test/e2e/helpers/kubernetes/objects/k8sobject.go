// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package k8sobject

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const DefaultWaitTimeout = 5 * time.Minute

var WaitTimeout = DefaultWaitTimeout

func Expect[T client.Object](name, namespace string, matcher func(T) bool) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		obj := emptyObject[T]()
		require.NoError(t, envConfig.Client().Resources().Get(ctx, name, namespace, obj))
		assert.True(t, matcher(obj))

		return ctx
	}
}

func Eventually[T client.Object](name, namespace string, matcher func(T) bool) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		obj := emptyObject[T]()
		obj.SetName(name)
		obj.SetNamespace(namespace)
		err := wait.For(conditions.New(envConfig.Client().Resources()).ResourceMatch(obj, func(o k8s.Object) bool {
			return matcher(o.(T))
		}), wait.WithImmediate(), wait.WithTimeout(WaitTimeout))
		require.NoError(t, err)

		return ctx
	}
}

func Create(obj client.Object) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, envConfig.Client().Resources().Create(ctx, obj))

		return ctx
	}
}

func Delete(obj client.Object) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, client.IgnoreNotFound(envConfig.Client().Resources().Delete(ctx, obj)))

		return WaitForDeletion(obj)(ctx, t, envConfig)
	}
}

func WaitForDeletion(obj client.Object) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := wait.For(
			conditions.New(envConfig.Client().Resources()).ResourceDeleted(obj),
			wait.WithImmediate(), wait.WithTimeout(WaitTimeout),
		)
		require.NoError(t, err)

		return ctx
	}
}

func emptyObject[T client.Object]() T {
	return reflect.New(reflect.TypeFor[T]().Elem()).Interface().(T)
}
