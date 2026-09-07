// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// Package v1beta6 contains API Schema definitions for the dynatrace v1beta6 API group
// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
package latest

import (
	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "dynatrace.com", Version: "v1beta6"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&latest.DynaKube{}, &latest.DynaKubeList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)

	return nil
}
