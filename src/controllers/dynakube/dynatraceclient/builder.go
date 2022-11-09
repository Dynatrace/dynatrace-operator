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

type Builder interface {
	SetContext(ctx context.Context) Builder
	SetDynakube(dynakube dynatracev1beta1.DynaKube) Builder
	SetTokens(tokens token.Tokens) Builder
	Build() (dtclient.Client, error)
	BuildWithTokenVerification(dynaKubeStatus *dynatracev1beta1.DynaKubeStatus) (dtclient.Client, error)
}

type builder struct {
	ctx       context.Context
	apiReader client.Reader
	dynakube  dynatracev1beta1.DynaKube
	tokens    token.Tokens
}

func NewBuilder(apiReader client.Reader) Builder {
	return builder{
		apiReader: apiReader,
	}
}

func (dynatraceClientBuilder builder) SetContext(ctx context.Context) Builder {
	dynatraceClientBuilder.ctx = ctx
	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builder) SetDynakube(dynakube dynatracev1beta1.DynaKube) Builder {
	dynatraceClientBuilder.dynakube = dynakube
	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builder) SetTokens(tokens token.Tokens) Builder {
	dynatraceClientBuilder.tokens = tokens
	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builder) context() context.Context {
	if dynatraceClientBuilder.ctx == nil {
		dynatraceClientBuilder.ctx = context.Background()
	}

	return dynatraceClientBuilder.ctx
}

func (dynatraceClientBuilder builder) getTokens() token.Tokens {
	if dynatraceClientBuilder.tokens == nil {
		dynatraceClientBuilder.tokens = token.Tokens{}
	}

	return dynatraceClientBuilder.tokens
}

// Build creates a new Dynatrace client using the settings configured on the given instance.
func (dynatraceClientBuilder builder) Build() (dtclient.Client, error) {
	namespace := dynatraceClientBuilder.dynakube.Namespace
	apiReader := dynatraceClientBuilder.apiReader

	opts := newOptions(dynatraceClientBuilder.context())
	opts.appendCertCheck(dynatraceClientBuilder.dynakube.Spec.SkipCertCheck)
	opts.appendNetworkZone(dynatraceClientBuilder.dynakube.Spec.NetworkZone)
	opts.appendDisableHostsRequests(dynatraceClientBuilder.dynakube.FeatureDisableHostsRequests())

	err := opts.appendProxySettings(apiReader, dynatraceClientBuilder.dynakube.Spec.Proxy, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = opts.appendTrustedCerts(apiReader, dynatraceClientBuilder.dynakube.Spec.TrustedCAs, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	apiToken := dynatraceClientBuilder.getTokens().ApiToken().Value
	paasToken := dynatraceClientBuilder.getTokens().PaasToken().Value

	if paasToken == "" {
		paasToken = apiToken
	}

	return dtclient.NewClient(dynatraceClientBuilder.dynakube.Spec.APIURL, apiToken, paasToken, opts.Opts...)
}

func (dynatraceClientBuilder builder) BuildWithTokenVerification(dynaKubeStatus *dynatracev1beta1.DynaKubeStatus) (dtclient.Client, error) {
	dynatraceClient, err := dynatraceClientBuilder.Build()
	if err != nil {
		return nil, err
	}

	err = dynatraceClientBuilder.getTokens().VerifyValues()
	if err != nil {
		return nil, err
	}

	dynatraceClientBuilder.tokens = dynatraceClientBuilder.getTokens().SetScopesForDynakube(dynatraceClientBuilder.dynakube)
	err = dynatraceClientBuilder.verifyTokenScopes(dynatraceClient, dynaKubeStatus)

	if err != nil {
		return nil, err
	}

	return dynatraceClient, nil
}

func (dynatraceClientBuilder builder) verifyTokenScopes(dynatraceClient dtclient.Client, dynaKubeStatus *dynatracev1beta1.DynaKubeStatus) error {
	if dynaKubeStatus.LastAPITokenProbeTimestamp == nil {
		dynaKubeStatus.LastAPITokenProbeTimestamp = &metav1.Time{}
	}

	if isLastApiCallTooRecent(dynaKubeStatus) {
		log.Info("returning a cached result because tokens are only validated once every five minutes to avoid rate limiting")
		err := lastErrorFromCondition(dynaKubeStatus)

		if err != nil {
			return err
		}
	} else {
		dynaKubeStatus.LastAPITokenProbeTimestamp = address.Of(metav1.Now())
		err := dynatraceClientBuilder.tokens.VerifyScopes(dynatraceClient)

		if err != nil {
			return err
		}
	}

	return nil
}

func lastErrorFromCondition(dynaKubeStatus *dynatracev1beta1.DynaKubeStatus) error {
	oldCondition := meta.FindStatusCondition(dynaKubeStatus.Conditions, dynatracev1beta1.TokenConditionType)
	if oldCondition != nil && oldCondition.Reason != dynatracev1beta1.ReasonTokenReady {
		return errors.New(oldCondition.Message)
	}

	return nil
}

func isLastApiCallTooRecent(dynaKubeStatus *dynatracev1beta1.DynaKubeStatus) bool {
	return time.Now().Before(dynaKubeStatus.LastAPITokenProbeTimestamp.Add(apiTokenProbeDelay))
}
