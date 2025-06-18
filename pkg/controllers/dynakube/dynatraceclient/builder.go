package dynatraceclient

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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
	SetDynakube(dk dynakube.DynaKube) Builder
	SetTokens(tokens token.Tokens) Builder
	Build() (dtclient.Client, error)
	BuildWithTokenVerification(dkStatus *dynakube.DynaKubeStatus) (dtclient.Client, error)
}

type builder struct {
	ctx       context.Context
	apiReader client.Reader
	tokens    token.Tokens
	dk        dynakube.DynaKube
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

func (dynatraceClientBuilder builder) SetDynakube(dk dynakube.DynaKube) Builder {
	dynatraceClientBuilder.dk = dk

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
	namespace := dynatraceClientBuilder.dk.Namespace
	apiReader := dynatraceClientBuilder.apiReader

	opts := newOptions(dynatraceClientBuilder.context())
	opts.appendCertCheck(dynatraceClientBuilder.dk.Spec.SkipCertCheck)
	opts.appendNetworkZone(dynatraceClientBuilder.dk.Spec.NetworkZone)
	opts.appendHostGroup(dynatraceClientBuilder.dk.OneAgent().GetHostGroup())

	err := opts.appendProxySettings(apiReader, &dynatraceClientBuilder.dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = opts.appendTrustedCerts(apiReader, dynatraceClientBuilder.dk.Spec.TrustedCAs, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	apiToken := dynatraceClientBuilder.getTokens().APIToken().Value
	paasToken := dynatraceClientBuilder.getTokens().PaasToken().Value

	if paasToken == "" {
		paasToken = apiToken
	}

	return dtclient.NewClient(dynatraceClientBuilder.dk.Spec.APIURL, apiToken, paasToken, opts.Opts...)
}

func (dynatraceClientBuilder builder) BuildWithTokenVerification(dkStatus *dynakube.DynaKubeStatus) (dtclient.Client, error) {
	dynatraceClient, err := dynatraceClientBuilder.Build()
	if err != nil {
		return nil, err
	}

	err = dynatraceClientBuilder.getTokens().VerifyValues()
	if err != nil {
		return nil, err
	}

	dynatraceClientBuilder.tokens = dynatraceClientBuilder.getTokens().AddFeatureScopesToTokens()

	err = dynatraceClientBuilder.verifyTokenScopes(dynatraceClient, dkStatus)
	if err != nil {
		return nil, err
	}

	return dynatraceClient, nil
}

func (dynatraceClientBuilder builder) verifyTokenScopes(dynatraceClient dtclient.Client, dkStatus *dynakube.DynaKubeStatus) error {
	if !dynatraceClientBuilder.dk.IsTokenScopeVerificationAllowed(timeprovider.New()) {
		log.Info(dynakube.GetCacheValidMessage(
			"token verification",
			dkStatus.DynatraceAPI.LastTokenScopeRequest,
			dynatraceClientBuilder.dk.APIRequestThreshold()))

		return lastErrorFromCondition(dkStatus)
	}

	err := dynatraceClientBuilder.tokens.VerifyScopes(dynatraceClientBuilder.ctx, dynatraceClient, dynatraceClientBuilder.dk)
	if err != nil {
		return err
	}

	log.Info("token verified")

	dkStatus.DynatraceAPI.LastTokenScopeRequest = metav1.Now()

	return nil
}

func lastErrorFromCondition(dkStatus *dynakube.DynaKubeStatus) error {
	oldCondition := meta.FindStatusCondition(dkStatus.Conditions, dynakube.TokenConditionType)
	if oldCondition != nil && oldCondition.Reason != dynakube.ReasonTokenReady {
		return errors.New(oldCondition.Message)
	}

	return nil
}
