package k8sconditions

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsOutdated determines if a given is considered outdated according to the DynaKube's FeatureApiRequestThreshold
// This is used for those conditions that are (also) used for limiting API requests.
func IsOutdated(timeProvider *timeprovider.Provider, dk *dynakube.DynaKube, conditionType string) bool {
	condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
	if condition == nil {
		return true
	}

	return (condition.Status == metav1.ConditionFalse && condition.Reason != OptionalScopeMissingReason) || timeProvider.IsOutdated(&condition.LastTransitionTime, dk.APIRequestThreshold())
}
