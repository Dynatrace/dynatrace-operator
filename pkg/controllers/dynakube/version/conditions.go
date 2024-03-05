package version

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	verifiedReason            = "Verified"
	verificationSkippedReason = "VerificationSkipped"
)

func setVerifiedCondition(conditions *[]metav1.Condition, conditionType string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  verifiedReason,
		Message: "Version verified for component.",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

func setVerificationSkippedReasonCondition(conditions *[]metav1.Condition, conditionType string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  verificationSkippedReason,
		Message: "Version verification skipped, due to custom setup.",
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
