package dynatraceclient

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

	err := opts.appendProxySettings(apiReader, &dynatraceClientBuilder.dynakube)
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
	if !dynatraceClientBuilder.dynakube.IsTokenScopeVerificationAllowed(timeprovider.New()) {
		log.Info(dynatracev1beta1.GetCacheValidMessage(
			"token verification",
			dynaKubeStatus.DynatraceApi.LastTokenScopeRequest,
			dynatraceClientBuilder.dynakube.FeatureApiRequestThreshold()))
		return lastErrorFromCondition(dynaKubeStatus)
	}

	err := dynatraceClientBuilder.tokens.VerifyScopes(dynatraceClient)
	if err != nil {
		return err
	}

	log.Info("token verified")
	dynaKubeStatus.DynatraceApi.LastTokenScopeRequest = metav1.Now()
	return nil
}

func lastErrorFromCondition(dynaKubeStatus *dynatracev1beta1.DynaKubeStatus) error {
	oldCondition := meta.FindStatusCondition(dynaKubeStatus.Conditions, dynatracev1beta1.TokenConditionType)
	if oldCondition != nil && oldCondition.Reason != dynatracev1beta1.ReasonTokenReady {
		return errors.New(oldCondition.Message)
	}

	return nil
}
