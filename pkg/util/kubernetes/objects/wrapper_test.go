// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sobject

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestRetryCreateOrUpate(t *testing.T) {
	t.Run("on conflict", func(t *testing.T) {
		obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
		var calls int
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				var err error
				if calls == 0 {
					err = k8serrors.NewConflict(schema.GroupResource{}, obj.GetName(), errors.New("boom"))
				}
				calls++

				return err
			},
		}, obj)

		_, err := RetryCreateOrUpdate(t.Context(), c, obj, func() error {
			obj.Data = map[string]string{"foo": "bar"}

			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 2, calls)
	})

	t.Run("on generic error", func(t *testing.T) {
		obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
		expectErr := k8serrors.NewAlreadyExists(schema.GroupResource{}, obj.GetName())

		var calls int
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				var err error
				if calls == 0 {
					err = expectErr
				}
				calls++

				return err
			},
		}, obj)

		_, err := RetryCreateOrUpdate(t.Context(), c, obj, func() error {
			obj.Data = map[string]string{"foo": "bar"}

			return nil
		})

		require.ErrorIs(t, err, expectErr)
		require.Equal(t, 1, calls)
	})
}
