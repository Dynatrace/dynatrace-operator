package dynakube

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TokenConditionType identifies the token validity condition.
	TokenConditionType string = "Tokens"

	// APITokenConditionType identifies the API Token validity condition.
	APITokenConditionType string = "APIToken"

	// PaaSTokenConditionType identifies the PaaS Token validity condition.
	PaaSTokenConditionType string = "PaaSToken"

	// DataIngestTokenConditionType identifies the DataIngest Token validity condition.
	DataIngestTokenConditionType string = "DataIngestToken"
)

// Possible reasons for ApiToken and PaaSToken conditions.
const (
	// ReasonTokenReady is set when a token has passed verifications.
	ReasonTokenReady string = "TokenReady"

	// ReasonTokenError is set when an unknown error has been found when verifying the token.
	ReasonTokenError string = "TokenError"

	ReasonCreated         string = "Created"
	ReasonError           string = "Error"
	ReasonUnexpectedError string = "UnexpectedError"
	ReasonUpToDate        string = "UpToDate"
)

// ActiveGate related conditions.
const (
	ActiveGateConnectionInfoConditionType string = "ActiveGateConnectionInfo"
	ActiveGateStatefulSetConditionType    string = "ActiveGateStatefulSet"
	ActiveGateVersionConditionType        string = "ActiveGateVersion"
)

func (dynakube *DynaKube) SetActiveGateConnectionInfoCondition(err error) error {
	if err != nil {
		meta.SetStatusCondition(&dynakube.Status.Conditions, metav1.Condition{
			Type:    ActiveGateConnectionInfoConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonUnexpectedError,
			Message: err.Error(),
		})

		return err
	}

	meta.SetStatusCondition(&dynakube.Status.Conditions, metav1.Condition{
		Type:   ActiveGateConnectionInfoConditionType,
		Status: metav1.ConditionTrue,
		Reason: ReasonCreated,
	})

	return err
}

func (dynakube *DynaKube) SetActiveGateStatefulSetErrorCondition(err error) error {
	if err != nil {
		meta.SetStatusCondition(&dynakube.Status.Conditions, metav1.Condition{
			Type:    ActiveGateStatefulSetConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonError,
			Message: err.Error(),
		})

		return err
	}

	return err
}

func (dynakube *DynaKube) SetActiveGateVersionCondition(err error) error {
	if err != nil {
		meta.SetStatusCondition(&dynakube.Status.Conditions, metav1.Condition{
			Type:    ActiveGateVersionConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonError,
			Message: err.Error(),
		})

		return err
	}

	meta.SetStatusCondition(&dynakube.Status.Conditions, metav1.Condition{
		Type:    ActiveGateVersionConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonUpToDate,
		Message: dynakube.Status.ActiveGate.VersionStatus.ImageID,
	})

	return nil
}
