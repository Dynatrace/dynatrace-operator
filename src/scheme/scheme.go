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

package scheme

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1"
	_ "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/dynakube"
	_ "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	_ "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// Scheme contains the type definitions used by the Operator and tests
var Scheme = k8sruntime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(dynatracev1alpha1.AddToScheme(Scheme))
	utilruntime.Must(dynatracev1beta1.AddToScheme(Scheme))
	utilruntime.Must(istiov1alpha3.AddToScheme(Scheme))
	utilruntime.Must(corev1.AddToScheme(Scheme))
	utilruntime.Must(apiv1.AddToScheme(Scheme))
	// +kubebuilder:scaffold:scheme
}
