package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TokenReadyConditionMessage             = "Token ready"
	TokenWithoutDataIngestConditionMessage = "Token ready, DataIngest token not provided"
	TokenVerificationConditionMessage      = "Token verification failed"
	TokenNotFoundConditionMessage          = "Token secret not found"
)

func (controller *Controller) setConditionTokenError(dk *dynakube.DynaKube, err error) {
	var msg string

	switch {
	case k8serrors.IsNotFound(err):
		msg = TokenNotFoundConditionMessage
	default:
		msg = TokenVerificationConditionMessage
	}

	tokenErrorCondition := metav1.Condition{
		Type:    dynakube.TokenConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  dynakube.ReasonTokenError,
		Message: msg,
	}

	log.Error(err, "token verification failed", "dynakube", dk.Name, "namespace", dk.Namespace)
	controller.setCondition(dk, tokenErrorCondition)
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

	controller.setCondition(dk, tokenErrorCondition)
}

// TODO: Probably should be removed, as most of this is done inside meta.SetStatusCondition the removeDeprecatedConditionTypes already did its job, as it has been in since forever
func (controller *Controller) setCondition(dk *dynakube.DynaKube, newCondition metav1.Condition) {
	controller.removeDeprecatedConditionTypes(dk)
	statusCondition := meta.FindStatusCondition(dk.Status.Conditions, newCondition.Type)

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
