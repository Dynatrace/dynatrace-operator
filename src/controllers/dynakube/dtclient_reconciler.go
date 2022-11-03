package dynakube

import (
	"context"
	"errors"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DynatraceClientReconciler struct {
	Client              client.Client
	DynatraceClientFunc DynatraceClientFunc
	Tokens              token.Tokens
}

func NewDynatraceClientReconciler(client client.Client, dtClientFunc DynatraceClientFunc) *DynatraceClientReconciler {
	return &DynatraceClientReconciler{
		Client:              client,
		DynatraceClientFunc: dtClientFunc,
	}
}

func (r *DynatraceClientReconciler) Reconcile(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (dtclient.Client, error) {
	tokenReader := token.NewReader(r.Client, dynakube)
	tokens, err := tokenReader.ReadTokens(ctx)

	if err != nil {
		r.setConditionTokenSecretMissing(dynakube, err)
		return nil, err
	}

	err = tokens.VerifyValues()

	if err != nil {
		r.setConditionTokensHaveInvalidValues(dynakube, err)
		return nil, err
	}

	dynatraceClientFunc := r.DynatraceClientFunc
	if dynatraceClientFunc == nil {
		dynatraceClientFunc = BuildDynatraceClient
	}

	dynatraceClient, err := dynatraceClientFunc(NewDynatraceClientProperties(ctx, r.Client, *dynakube, tokens))

	if err != nil {
		r.setConditionDtcError(dynakube, err)
		return nil, err
	}

	if dynakube.Status.LastAPITokenProbeTimestamp == nil {
		dynakube.Status.LastAPITokenProbeTimestamp = &metav1.Time{}
	}

	if time.Now().Before(dynakube.Status.LastAPITokenProbeTimestamp.Add(5 * time.Second)) {
		oldCondition := meta.FindStatusCondition(dynakube.Status.Conditions, dynatracev1beta1.TokenConditionType)
		if oldCondition.Reason != dynatracev1beta1.ReasonTokenReady {
			return nil, errors.New("tokens are not valid")
		}
	} else {
		tokens = tokens.SetScopes(*dynakube)
		err = tokens.VerifyScopes(dynatraceClient)

		if err != nil {
			r.setConditionTokenIsMissingScopes(dynakube, err)
			return nil, err
		}
	}

	r.Tokens = tokens
	r.setConditionTokenReady(dynakube)
	dynakube.Status.LastAPITokenProbeTimestamp = address.Of(metav1.Now())

	return dynatraceClient, nil
}

func (r *DynatraceClientReconciler) setConditionTokenSecretMissing(dynakube *dynatracev1beta1.DynaKube, err error) {
	missingTokenCondition := metav1.Condition{
		Type:    dynatracev1beta1.TokenConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  dynatracev1beta1.ReasonTokenMissing,
		Message: err.Error(),
	}

	r.setAndLogCondition(dynakube, missingTokenCondition)
}

func (r *DynatraceClientReconciler) setConditionTokensHaveInvalidValues(dynakube *dynatracev1beta1.DynaKube, err error) {
	invalidValueCondition := metav1.Condition{
		Type:    dynatracev1beta1.TokenConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  dynatracev1beta1.ReasonTokenSecretInvalid,
		Message: err.Error(),
	}

	r.setAndLogCondition(dynakube, invalidValueCondition)
}

func (r *DynatraceClientReconciler) setConditionDtcError(dynakube *dynatracev1beta1.DynaKube, err error) {
	dynatraceClientErrorCondition := metav1.Condition{
		Type:    dynatracev1beta1.TokenConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  dynatracev1beta1.ReasonDynatraceClientError,
		Message: err.Error(),
	}

	r.setAndLogCondition(dynakube, dynatraceClientErrorCondition)
}

func (r *DynatraceClientReconciler) setConditionTokenIsMissingScopes(dynakube *dynatracev1beta1.DynaKube, err error) {
	missingScopeCondition := metav1.Condition{
		Type:    dynatracev1beta1.TokenConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  dynatracev1beta1.ReasonTokenScopeMissing,
		Message: err.Error(),
	}

	r.setAndLogCondition(dynakube, missingScopeCondition)
}

func (r *DynatraceClientReconciler) setConditionTokenReady(dynakube *dynatracev1beta1.DynaKube) {
	tokenValidCondition := metav1.Condition{
		Type:    dynatracev1beta1.TokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1beta1.ReasonTokenReady,
		Message: "tokens have been successfully validated",
	}

	r.setAndLogCondition(dynakube, tokenValidCondition)
}

func (r *DynatraceClientReconciler) setAndLogCondition(dynakube *dynatracev1beta1.DynaKube, newCondition metav1.Condition) {
	r.removeOldConditionTypes(dynakube)
	statusCondition := meta.FindStatusCondition(dynakube.Status.Conditions, newCondition.Type)

	if newCondition.Reason != dynatracev1beta1.ReasonTokenReady {
		log.Info("problem with token detected", "dynakube", dynakube.Name, "token", newCondition.Type,
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

func (r *DynatraceClientReconciler) removeOldConditionTypes(dynakube *dynatracev1beta1.DynaKube) {
	if meta.FindStatusCondition(dynakube.Status.Conditions, dynatracev1beta1.PaaSTokenConditionType) != nil {
		meta.RemoveStatusCondition(&dynakube.Status.Conditions, dynatracev1beta1.PaaSTokenConditionType)
	}
	if meta.FindStatusCondition(dynakube.Status.Conditions, dynatracev1beta1.APITokenConditionType) != nil {
		meta.RemoveStatusCondition(&dynakube.Status.Conditions, dynatracev1beta1.APITokenConditionType)
	}
	if meta.FindStatusCondition(dynakube.Status.Conditions, dynatracev1beta1.DataIngestTokenConditionType) != nil {
		meta.RemoveStatusCondition(&dynakube.Status.Conditions, dynatracev1beta1.DataIngestTokenConditionType)
	}
}
