package dynatraceclient

import (
	"context"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const apiTokenProbeDelay = 5 * time.Minute

type BuildFunc func(properties Properties) (dtclient.Client, error)

type Factory struct {
	ctx                 context.Context
	client              client.Client
	dynatraceClientFunc BuildFunc
}

func NewFactory(client client.Client, dtClientFunc BuildFunc) *Factory {
	return &Factory{
		client:              client,
		dynatraceClientFunc: dtClientFunc,
	}
}

func (r *Factory) Create(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (dtclient.Client, error) {
	r.ctx = ctx
	tokenReader := token.NewReader(r.client, dynakube)
	tokens, err := tokenReader.ReadTokens(ctx)

	if err != nil {
		return nil, err
	}

	err = tokens.VerifyValues()

	if err != nil {
		return nil, err
	}

	dynatraceClient, err := r.buildDynatraceClient(NewProperties(ctx, r.client, *dynakube, tokens))

	if err != nil {
		return nil, err
	}

	if dynakube.Status.LastAPITokenProbeTimestamp == nil {
		dynakube.Status.LastAPITokenProbeTimestamp = &metav1.Time{}
	}

	_, err = r.checkTokenScopes(dynakube, tokens, dynatraceClient)
	if err != nil {
		return nil, err
	}

	return dynatraceClient, nil
}

func (r *Factory) buildDynatraceClient(dynatraceClientProperties Properties) (dtclient.Client, error) {
	dynatraceClientFunc := r.dynatraceClientFunc
	if dynatraceClientFunc == nil {
		dynatraceClientFunc = BuildDynatraceClient
	}

	return dynatraceClientFunc(dynatraceClientProperties)
}

func (r *Factory) checkTokenScopes(dynakube *dynatracev1beta1.DynaKube, tokens token.Tokens, dynatraceClient dtclient.Client) (token.Tokens, error) {
	if isLastApiCallTooRecent(dynakube) {
		log.Info("returning a cached result because tokens are only validated once every five minutes to avoid rate limiting")
		err := lastErrorFromCondition(dynakube)

		if err != nil {
			return tokens, err
		}
	} else {
		dynakube.Status.LastAPITokenProbeTimestamp = address.Of(metav1.Now())
		tokens = tokens.SetScopesForDynakube(*dynakube)
		err := tokens.VerifyScopes(dynatraceClient)

		if err != nil {
			return tokens, err
		}
	}

	return tokens, nil
}

func lastErrorFromCondition(dynakube *dynatracev1beta1.DynaKube) error {
	oldCondition := meta.FindStatusCondition(dynakube.Status.Conditions, dynatracev1beta1.TokenConditionType)
	if oldCondition != nil && oldCondition.Reason != dynatracev1beta1.ReasonTokenReady {
		return errors.New(oldCondition.Message)
	}

	return nil
}

func isLastApiCallTooRecent(dynakube *dynatracev1beta1.DynaKube) bool {
	return time.Now().Before(dynakube.Status.LastAPITokenProbeTimestamp.Add(apiTokenProbeDelay))
}
