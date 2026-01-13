/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// NewClient returns a new controller-runtime fake Client configured with the Operator's scheme, and initialized with objs.
func NewClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).WithStatusSubresource(objs...).Build()
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
		&v1.CustomResourceDefinition{},
	}

	for _, object := range objects {
		clientBuilder.WithIndex(object, "metadata.name", func(o client.Object) []string {
			return []string{o.GetName()}
		})
	}

	return clientBuilder.Build()
}

func NewClientWithInterceptors(funcs interceptor.Funcs) client.Client {
	clientBuilder := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithInterceptorFuncs(funcs)

	return clientBuilder.Build()
}

func NewClientWithInterceptorsAndObjects(funcs interceptor.Funcs, objs ...client.Object) client.Client {
	clientBuilder := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(objs...).
		WithStatusSubresource(objs...).
		WithInterceptorFuncs(funcs)

	return clientBuilder.Build()
}
