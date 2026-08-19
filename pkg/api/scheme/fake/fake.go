// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package fake

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// NewClient returns a new controller-runtime fake Client configured with the Operator's scheme, and initialized with objs.
func NewClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).WithStatusSubresource(objs...).Build()
}

// NewClientWithManagedFields returns a fake client that populates the managed fields of returned objects.
func NewClientWithManagedFields(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).WithStatusSubresource(objs...).WithReturnManagedFields().Build()
}

// NewClientWithIndex returns a fake client with common indexes already configured.
func NewClientWithIndex(objs ...client.Object) client.Client {
	clientBuilder := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(objs...).
		WithStatusSubresource(objs...)

	objects := []runtime.Object{
		&corev1.Namespace{},
		&corev1.Secret{},
		&admissionregistrationv1.MutatingWebhookConfiguration{},
		&admissionregistrationv1.ValidatingWebhookConfiguration{},
		&apiextensionsv1.CustomResourceDefinition{},
	}

	for _, object := range objects {
		clientBuilder.WithIndex(object, "metadata.name", func(o client.Object) []string {
			return []string{o.GetName()}
		})
	}

	return clientBuilder.Build()
}

func NewClientWithInterceptors(funcs interceptor.Funcs, objs ...client.Object) client.Client {
	clientBuilder := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithInterceptorFuncs(funcs).
		WithObjects(objs...).
		WithStatusSubresource(objs...)

	return clientBuilder.Build()
}

func NewClientWithInterceptorsAndIndex(funcs interceptor.Funcs, objs ...client.Object) client.Client {
	clientBuilder := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithInterceptorFuncs(funcs).
		WithObjects(objs...).
		WithStatusSubresource(objs...)

	objects := []runtime.Object{
		&corev1.Namespace{},
		&corev1.Secret{},
		&admissionregistrationv1.MutatingWebhookConfiguration{},
		&admissionregistrationv1.ValidatingWebhookConfiguration{},
		&apiextensionsv1.CustomResourceDefinition{},
	}

	for _, object := range objects {
		clientBuilder.WithIndex(object, "metadata.name", func(o client.Object) []string {
			return []string{o.GetName()}
		})
	}

	return clientBuilder.Build()
}
