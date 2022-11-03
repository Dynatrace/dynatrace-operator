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
	Client                               client.Client
	DynatraceClientFunc                  DynatraceClientFunc
	Now                                  metav1.Time
	ApiToken, PaasToken, DataIngestToken string
	ValidTokens                          bool
	dkName, ns, secretKey                string
	status                               *dynatracev1beta1.DynaKubeStatus
}

func NewDynatraceClientReconciler(client client.Client, dtClientFunc DynatraceClientFunc) *DynatraceClientReconciler {
	return &DynatraceClientReconciler{
		Client:              client,
		DynatraceClientFunc: dtClientFunc,
	}
}

func (r *DynatraceClientReconciler) Reconcile(ctx context.Context, dynaKube *dynatracev1beta1.DynaKube) (dtclient.Client, error) {
	tokenReader := token.NewReader(r.Client, dynaKube)
	tokens, err := tokenReader.ReadTokens(ctx)

	if err != nil {
		// r.setConditionTokenSecretMissing(err)
		return nil, err
	}

	err = tokens.VerifyValues()

	if err != nil {
		// r.setConditionTokensHaveInvalidValues(err)
		return nil, err
	}

	dynatraceClientFunc := r.DynatraceClientFunc
	if dynatraceClientFunc == nil {
		dynatraceClientFunc = BuildDynatraceClient
	}

	dynatraceClient, err := dynatraceClientFunc(NewDynatraceClientProperties(ctx, r.Client, *dynaKube, tokens))

	if err != nil {
		// r.setConditionDtcError(err)
		return nil, err
	}

	if time.Now().Before(dynaKube.Status.LastAPITokenProbeTimestamp.Add(5 * time.Minute)) {
		oldCondition := meta.FindStatusCondition(r.status.Conditions, dynatracev1beta1.APITokenConditionType)
		if oldCondition.Reason != dynatracev1beta1.ReasonTokenReady {
			return nil, errors.New("tokens are not valid")
		}
	} else {
		err = tokens.VerifyScopes(dynatraceClient)

		if err != nil {
			// r.setConditionTokenIsMissingScopes(err)
			return nil, err
		}
	}

	dynaKube.Status.LastAPITokenProbeTimestamp = address.Of(metav1.Now())

	return dynatraceClient, nil
}

func (r *DynatraceClientReconciler) setConditionTokenSecretMissing(err error) {

}

func (r *DynatraceClientReconciler) removePaaSTokenCondition() {
	if meta.FindStatusCondition(r.status.Conditions, dynatracev1beta1.PaaSTokenConditionType) != nil {
		meta.RemoveStatusCondition(&r.status.Conditions, dynatracev1beta1.PaaSTokenConditionType)
	}
}

func (r *DynatraceClientReconciler) setAndLogCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	c := meta.FindStatusCondition(*conditions, condition.Type)

	if condition.Reason != dynatracev1beta1.ReasonTokenReady {
		r.ValidTokens = false
		log.Info("problem with token detected", "dynakube", r.dkName, "token", condition.Type,
			"msg", condition.Message)
	}

	if c != nil && c.Reason == condition.Reason && c.Message == condition.Message && c.Status == condition.Status {
		return
	}

	condition.LastTransitionTime = r.Now
	meta.SetStatusCondition(conditions, condition)
}
