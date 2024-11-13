package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SettingsExistReason = "LogMonSettingsExist"
)

func SetLogMonitoringSettingExists(conditions *[]metav1.Condition, conditionType string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  SettingsExistReason,
		Message: "LogMonitoring settings already exist, will not create new ones.",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
