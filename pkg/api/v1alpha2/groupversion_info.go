// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// Package v1alpha2 contains API Schema definitions for the dynatrace v1alpha2 API group
// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
package v1alpha2

import (
	v1alpha2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "dynatrace.com", Version: "v1alpha2"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&v1alpha2.EdgeConnect{}, &v1alpha2.EdgeConnectList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)

	return nil
}
