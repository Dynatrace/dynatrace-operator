package k8sconditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConfigMapCreatedReason          = "ConfigMapCreated"
	ConfigMapUpdatedReason          = "ConfigMapUpdated"
	ConfigMapCreatedOrUpdatedReason = "ConfigMapCreatedOrUpdated"
	ConfigMapOutdatedReason         = "ConfigMapOutdated"
	ConfigMapGenerationFailed       = "ConfigMapGenerationFailed"
)

func SetConfigMapCreatedOrUpdated(conditions *[]metav1.Condition, conditionType, name string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  ConfigMapCreatedOrUpdatedReason,
		Message: appendCreatedOrUpdatedSuffix(name),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func SetConfigMapGenFailed(conditions *[]metav1.Condition, conditionType string, err error) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  ConfigMapGenerationFailed,
		Message: "Failed to generate configmap: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
