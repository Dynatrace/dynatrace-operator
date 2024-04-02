package version

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	downgradeReason           = "Downgrade"
	verifiedReason            = "Verified"
	verificationSkippedReason = "VerificationSkipped"
	verificationFailedReason  = "VerificationFailed"
)

func setDowngradeCondition(conditions *[]metav1.Condition, conditionType, previousVersion, newVersion string) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  downgradeReason,
		Message: fmt.Sprintf("Downgrade detected from %s to %s, which is not supported for this feature.", previousVersion, newVersion),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}

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

func setVerificationFailedReasonCondition(conditions *[]metav1.Condition, conditionType string, err error) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  verificationFailedReason,
		Message: "Version verification failed, due to: " + err.Error(),
	}
	_ = meta.SetStatusCondition(conditions, condition)
}
