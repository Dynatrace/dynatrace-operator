package latest

import (
	lts "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = lts.GroupVersion

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = lts.SchemeBuilder

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = lts.AddToScheme
)
