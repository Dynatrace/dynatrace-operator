package istio

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// This component was only used in the conditions, while another (`oneagent`) one was used in the creation of the ServiceEntries.
	// To avoid confusion and reduce complexity, we deprecate this component name, and just use the same component name everywhere.
	deprecatedComponent = "CodeModule"
)

// TODO: Remove this function in a future release
func migrateDeprecatedCondition(conditions *[]metav1.Condition) {
	if conditions == nil {
		return
	}

	depCondition := meta.FindStatusCondition(*conditions, getConditionTypeName(deprecatedComponent))
	if depCondition != nil {
		newConditionType := getConditionTypeName(CodeModuleComponent)
		depCondition.Type = newConditionType
		_ = meta.RemoveStatusCondition(conditions, depCondition.Type)
		_ = meta.SetStatusCondition(conditions, *depCondition)
	}
}
