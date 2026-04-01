package dynatraceclient

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BuilderV2 interface {
	SetDynakube(dk dynakube.DynaKube) BuilderV2
	SetTokens(tokens token.Tokens) BuilderV2
	SetUserAgentSuffix(suffix string) BuilderV2
	Build(ctx context.Context) (*dtclient.ClientV2, error)
}

type builderV2 struct {
	apiReader       client.Reader
	tokens          token.Tokens
	dk              dynakube.DynaKube
	userAgentSuffix string
}

func NewBuilderV2(apiReader client.Reader) BuilderV2 {
	return builderV2{
		apiReader: apiReader,
	}
}

func (dynatraceClientBuilder builderV2) SetDynakube(dk dynakube.DynaKube) BuilderV2 {
	dynatraceClientBuilder.dk = dk

	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builderV2) SetTokens(tokens token.Tokens) BuilderV2 {
	dynatraceClientBuilder.tokens = tokens

	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builderV2) SetUserAgentSuffix(suffx string) BuilderV2 {
	dynatraceClientBuilder.userAgentSuffix = suffx

	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builderV2) getTokens() token.Tokens {
	if dynatraceClientBuilder.tokens == nil {
		dynatraceClientBuilder.tokens = token.Tokens{}
	}

	return dynatraceClientBuilder.tokens
}

// Build creates a new Dynatrace client using the settings configured on the given instance.
func (dynatraceClientBuilder builderV2) Build(ctx context.Context) (*dtclient.ClientV2, error) {
	namespace := dynatraceClientBuilder.dk.Namespace
	apiReader := dynatraceClientBuilder.apiReader

	if dynatraceClientBuilder.dk.Spec.APIURL == "" {
		return nil, errors.New("url is empty")
	}

	opts := newOptionsV2(ctx)
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

	if apiToken == "" && paasToken == "" {
		return nil, errors.New("tokens are empty")
	}

	if paasToken == "" {
		paasToken = apiToken
	}

	opts.Opts = append(opts.Opts, dtclient.WithUserAgentSuffix(dynatraceClientBuilder.userAgentSuffix))

	opts.Opts = append(opts.Opts, dtclient.WithAPIToken(apiToken))
	opts.Opts = append(opts.Opts, dtclient.WithPaasToken(paasToken))

	return dtclient.NewClientV2(dynatraceClientBuilder.dk.Spec.APIURL, opts.Opts...)
}
