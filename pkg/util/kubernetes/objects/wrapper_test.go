// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sobject

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestRetryCreateOrUpate(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Data: map[string]string{"foo": "bar"}}
		var calls int
		c := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				calls++

				return nil
			},
		})

		err := RetryCreateOrUpdate(t.Context(), c, obj, func() error { return nil })

		require.NoError(t, err)
		require.Equal(t, 1, calls)
	})

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

		err := RetryCreateOrUpdate(t.Context(), c, obj, func() error {
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

		err := RetryCreateOrUpdate(t.Context(), c, obj, func() error {
			obj.Data = map[string]string{"foo": "bar"}

			return nil
		})

		require.ErrorIs(t, err, expectErr)
		require.Equal(t, 1, calls)
	})

	t.Run("missing scheme", func(t *testing.T) {
		obj := &UnregisteredObject{}
		err := RetryCreateOrUpdate(t.Context(), fake.NewClient(), obj, func() error { return nil })
		require.Truef(t, runtime.IsNotRegisteredError(err), "%v", err)
	})
}

func TestApplyStatus(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "bar", Namespace: "foo", Generation: 123}}
		c := fake.NewClientWithManagedFields(deploy.DeepCopy())
		deploy.Status.ObservedGeneration = 123
		require.NoError(t, ApplyStatus(t.Context(), c, deploy))
		got := &appsv1.Deployment{}
		require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(deploy), got))
		require.EqualValues(t, 123, got.Status.ObservedGeneration)
		require.Len(t, got.ManagedFields, 1)
		require.Equal(t, "dynatrace-operator", got.ManagedFields[0].Manager)
	})

	t.Run("missing scheme", func(t *testing.T) {
		obj := &UnregisteredObject{}
		err := ApplyStatus(t.Context(), fake.NewClient(), obj)
		require.Truef(t, runtime.IsNotRegisteredError(err), "%v", err)
	})
}

type UnregisteredObject struct{}

var _ client.Object = &UnregisteredObject{}

func (u *UnregisteredObject) DeepCopyObject() runtime.Object                { return u }
func (u *UnregisteredObject) GetAnnotations() map[string]string             { return nil }
func (u *UnregisteredObject) GetCreationTimestamp() metav1.Time             { return metav1.Time{} }
func (u *UnregisteredObject) GetDeletionGracePeriodSeconds() *int64         { return nil }
func (u *UnregisteredObject) GetDeletionTimestamp() *metav1.Time            { return nil }
func (u *UnregisteredObject) GetFinalizers() []string                       { return nil }
func (u *UnregisteredObject) GetGenerateName() string                       { return "" }
func (u *UnregisteredObject) GetGeneration() int64                          { return 0 }
func (u *UnregisteredObject) GetLabels() map[string]string                  { return nil }
func (u *UnregisteredObject) GetManagedFields() []metav1.ManagedFieldsEntry { return nil }
func (u *UnregisteredObject) GetName() string                               { return "" }
func (u *UnregisteredObject) GetNamespace() string                          { return "" }
func (u *UnregisteredObject) GetObjectKind() schema.ObjectKind              { return schema.EmptyObjectKind }
func (u *UnregisteredObject) GetOwnerReferences() []metav1.OwnerReference   { return nil }
func (u *UnregisteredObject) GetResourceVersion() string                    { return "" }
func (u *UnregisteredObject) GetSelfLink() string                           { return "" }
func (u *UnregisteredObject) GetUID() types.UID                             { return "" }

func (u *UnregisteredObject) SetAnnotations(annotations map[string]string)               {}
func (u *UnregisteredObject) SetCreationTimestamp(timestamp metav1.Time)                 {}
func (u *UnregisteredObject) SetDeletionGracePeriodSeconds(*int64)                       {}
func (u *UnregisteredObject) SetDeletionTimestamp(timestamp *metav1.Time)                {}
func (u *UnregisteredObject) SetFinalizers(finalizers []string)                          {}
func (u *UnregisteredObject) SetGenerateName(name string)                                {}
func (u *UnregisteredObject) SetGeneration(generation int64)                             {}
func (u *UnregisteredObject) SetLabels(labels map[string]string)                         {}
func (u *UnregisteredObject) SetManagedFields(managedFields []metav1.ManagedFieldsEntry) {}
func (u *UnregisteredObject) SetName(name string)                                        {}
func (u *UnregisteredObject) SetNamespace(namespace string)                              {}
func (u *UnregisteredObject) SetOwnerReferences([]metav1.OwnerReference)                 {}
func (u *UnregisteredObject) SetResourceVersion(version string)                          {}
func (u *UnregisteredObject) SetSelfLink(selfLink string)                                {}
func (u *UnregisteredObject) SetUID(uid types.UID)                                       {}
