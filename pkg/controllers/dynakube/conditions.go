package dynakube

import (
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (controller *Controller) setConditionTokenError(dynakube *dynatracev1beta2.DynaKube, err error) {
	tokenErrorCondition := metav1.Condition{
		Type:    dynatracev1beta2.TokenConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  dynatracev1beta2.ReasonTokenError,
		Message: err.Error(),
	}

	controller.setAndLogCondition(dynakube, tokenErrorCondition)
}

func (controller *Controller) setConditionTokenReady(dynakube *dynatracev1beta2.DynaKube) {
	tokenErrorCondition := metav1.Condition{
		Type:   dynatracev1beta2.TokenConditionType,
		Status: metav1.ConditionTrue,
		Reason: dynatracev1beta2.ReasonTokenReady,
	}

	controller.setAndLogCondition(dynakube, tokenErrorCondition)
}

// TODO: Probably should be removed, as most of this is done inside meta.SetStatusCondition (except the logging) the removeDeprecatedConditionTypes already did its job, as it has been in since forever
func (controller *Controller) setAndLogCondition(dynakube *dynatracev1beta2.DynaKube, newCondition metav1.Condition) {
	controller.removeDeprecatedConditionTypes(dynakube)
	statusCondition := meta.FindStatusCondition(dynakube.Status.Conditions, newCondition.Type)

	if newCondition.Reason != dynatracev1beta2.ReasonTokenReady {
		log.Info("problem with token detected",
			"dynakube", dynakube.Name, "namespace", dynakube.Namespace,
			"token", newCondition.Type,
			"message", newCondition.Message)
	}

	if areStatusesEqual(statusCondition, newCondition) {
		return
	}

	newCondition.LastTransitionTime = metav1.Now()
	meta.SetStatusCondition(&dynakube.Status.Conditions, newCondition)
}

func areStatusesEqual(statusCondition *metav1.Condition, newCondition metav1.Condition) bool {
	return statusCondition != nil &&
		statusCondition.Reason == newCondition.Reason &&
		statusCondition.Message == newCondition.Message &&
		statusCondition.Status == newCondition.Status
}

func (controller *Controller) removeDeprecatedConditionTypes(dynakube *dynatracev1beta2.DynaKube) {
	if meta.FindStatusCondition(dynakube.Status.Conditions, dynatracev1beta2.PaaSTokenConditionType) != nil {
		meta.RemoveStatusCondition(&dynakube.Status.Conditions, dynatracev1beta2.PaaSTokenConditionType)
	}

	if meta.FindStatusCondition(dynakube.Status.Conditions, dynatracev1beta2.APITokenConditionType) != nil {
		meta.RemoveStatusCondition(&dynakube.Status.Conditions, dynatracev1beta2.APITokenConditionType)
	}

	if meta.FindStatusCondition(dynakube.Status.Conditions, dynatracev1beta2.DataIngestTokenConditionType) != nil {
		meta.RemoveStatusCondition(&dynakube.Status.Conditions, dynatracev1beta2.DataIngestTokenConditionType)
	}
}
