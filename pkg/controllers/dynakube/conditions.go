package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TokenReadyConditionMessage             = "Token ready"
	TokenWithoutDataIngestConditionMessage = "Token ready, DataIngest token not provided"
)

func (controller *Controller) setConditionTokenError(ctx context.Context, dk *dynakube.DynaKube, err error) {
	tokenErrorCondition := metav1.Condition{
		Type:    dynakube.TokenConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  dynakube.ReasonTokenError,
		Message: err.Error(),
	}

	controller.setAndLogCondition(ctx, dk, tokenErrorCondition)
}

func (controller *Controller) setConditionTokenReady(ctx context.Context, dk *dynakube.DynaKube, dataIngestTokenProvided bool) {
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

	controller.setAndLogCondition(ctx, dk, tokenErrorCondition)
}

// TODO: Probably should be removed, as most of this is done inside meta.SetStatusCondition (except the logging) the removeDeprecatedConditionTypes already did its job, as it has been in since forever
func (controller *Controller) setAndLogCondition(ctx context.Context, dk *dynakube.DynaKube, newCondition metav1.Condition) {
	log := logd.FromContext(ctx)

	controller.removeDeprecatedConditionTypes(dk)
	statusCondition := meta.FindStatusCondition(dk.Status.Conditions, newCondition.Type)

	if newCondition.Reason != dynakube.ReasonTokenReady {
		log.Warn("problem with token detected",
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
