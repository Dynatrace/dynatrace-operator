// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scheme

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest"
	_ "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	_ "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2"
	_ "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5"
	_ "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// Scheme contains the type definitions used by the Operator and tests.
var Scheme = k8sruntime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(v1alpha1.AddToScheme(Scheme))
	utilruntime.Must(v1alpha2.AddToScheme(Scheme))
	utilruntime.Must(v1beta5.AddToScheme(Scheme))
	utilruntime.Must(latest.AddToScheme(Scheme))
	utilruntime.Must(istiov1beta1.AddToScheme(Scheme))
	utilruntime.Must(corev1.AddToScheme(Scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(Scheme))
	// +kubebuilder:scaffold:scheme
}
