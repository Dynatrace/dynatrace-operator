package conditions

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsOptionalScopeAvailable(dk *dynakube.DynaKube, conditionType string) bool {
	condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
	if condition == nil {
		return false
	}

	return condition.Status == metav1.ConditionTrue
}
