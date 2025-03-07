package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TokenReadyConditionMessage             = "Token ready"
	TokenWithoutDataIngestConditionMessage = "Token ready, DataIngest token not provided"
)

func (controller *Controller) setConditionTokenError(dk *dynakube.DynaKube, err error) {
	tokenErrorCondition := metav1.Condition{
		Type:    dynakube.TokenConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  dynakube.ReasonTokenError,
		Message: err.Error(),
	}

	controller.setAndLogCondition(dk, tokenErrorCondition)
}

func (controller *Controller) setConditionTokenReady(dk *dynakube.DynaKube, dataIngestTokenProvided bool) {
	msg := TokenWithoutDataIngestConditionMessage
	if dataIngestTokenProvided {
		msg = TokenReadyConditionMessage
	}

	tokenErrorCondition := metav1.Condition{
		Type:    dynakube.TokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynakube.ReasonTokenReady,
		Message: msg,
	}

	controller.setAndLogCondition(dk, tokenErrorCondition)
}

// TODO: Probably should be removed, as most of this is done inside meta.SetStatusCondition (except the logging) the removeDeprecatedConditionTypes already did its job, as it has been in since forever
func (controller *Controller) setAndLogCondition(dk *dynakube.DynaKube, newCondition metav1.Condition) {
	controller.removeDeprecatedConditionTypes(dk)
	statusCondition := meta.FindStatusCondition(dk.Status.Conditions, newCondition.Type)

	if newCondition.Reason != dynakube.ReasonTokenReady {
		log.Info("problem with token detected",
			"dynakube", dk.Name, "namespace", dk.Namespace,
			"token", newCondition.Type,
			"message", newCondition.Message)
	}

	if areStatusesEqual(statusCondition, newCondition) {
		return
	}

	newCondition.LastTransitionTime = metav1.Now()
	meta.SetStatusCondition(&dk.Status.Conditions, newCondition)
}

func areStatusesEqual(statusCondition *metav1.Condition, newCondition metav1.Condition) bool {
	return statusCondition != nil &&
		statusCondition.Reason == newCondition.Reason &&
		statusCondition.Message == newCondition.Message &&
		statusCondition.Status == newCondition.Status
}

func (controller *Controller) removeDeprecatedConditionTypes(dk *dynakube.DynaKube) {
	if meta.FindStatusCondition(dk.Status.Conditions, dynakube.PaaSTokenConditionType) != nil {
		meta.RemoveStatusCondition(&dk.Status.Conditions, dynakube.PaaSTokenConditionType)
	}

	if meta.FindStatusCondition(dk.Status.Conditions, dynakube.APITokenConditionType) != nil {
		meta.RemoveStatusCondition(&dk.Status.Conditions, dynakube.APITokenConditionType)
	}

	if meta.FindStatusCondition(dk.Status.Conditions, dynakube.DataIngestTokenConditionType) != nil {
		meta.RemoveStatusCondition(&dk.Status.Conditions, dynakube.DataIngestTokenConditionType)
	}
}
