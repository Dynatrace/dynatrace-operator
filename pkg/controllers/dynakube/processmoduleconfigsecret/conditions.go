package processmoduleconfigsecret

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	conditionType = "ProcessModuleConfig"

	secretCreatedReason = "SecretCreated"
	secretUpdatedReason = "SecretUpdated"

	secretOutdatedReason    = "SecretOutdated"
	kubeApiErrorReason      = "KubeApiError"
	dynatraceApiErrorReason = "DynatraceApiError"
)

func (r *Reconciler) setSecretCreatedCondition() {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  secretCreatedReason,
		Message: "Process module config secret created",
	}
	_ = meta.SetStatusCondition(r.dynakube.Conditions(), condition)
}

func (r *Reconciler) setSecretUpdatedCondition() {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  secretUpdatedReason,
		Message: "Process module config secret updated",
	}
	_ = meta.SetStatusCondition(r.dynakube.Conditions(), condition)
}

func (r *Reconciler) setSecretOutdatedCondition() {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  secretOutdatedReason,
		Message: "Process module config secret has passed its TTL",
	}
	_ = meta.SetStatusCondition(r.dynakube.Conditions(), condition)
}

func (r *Reconciler) setKubeApiErrorCondition(err error) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  kubeApiErrorReason,
		Message: "A problem occurred when using the Kubernetes API: " + err.Error(),
	}
	_ = meta.SetStatusCondition(r.dynakube.Conditions(), condition)
}

func (r *Reconciler) setDynatraceApiErrorCondition(err error) {
	condition := metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  dynatraceApiErrorReason,
		Message: "A problem occurred when using the Dynatrace API: " + err.Error(),
	}
	_ = meta.SetStatusCondition(r.dynakube.Conditions(), condition)
}
