// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// Package v1alpha1 contains API Schema definitions for the dynatrace v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
package v1alpha1

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "dynatrace.com", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&edgeconnect.EdgeConnect{}, &edgeconnect.EdgeConnectList{},
		&dtprometheus.DTPrometheus{}, &dtprometheus.DTPrometheusList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)

	return nil
}
