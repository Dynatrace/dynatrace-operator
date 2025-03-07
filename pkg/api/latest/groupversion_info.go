package latest

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = v1beta3.GroupVersion

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = v1beta3.SchemeBuilder

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = v1beta3.AddToScheme
)
