package dynatraceclient

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Builder interface {
	SetDynakube(dk dynakube.DynaKube) Builder
	SetTokens(tokens token.Tokens) Builder
	Build(ctx context.Context) (dtclient.Client, error)
}

type builder struct {
	apiReader client.Reader
	tokens    token.Tokens
	dk        dynakube.DynaKube
}

func NewBuilder(apiReader client.Reader) Builder {
	return builder{
		apiReader: apiReader,
	}
}

func (dynatraceClientBuilder builder) SetDynakube(dk dynakube.DynaKube) Builder {
	dynatraceClientBuilder.dk = dk

	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builder) SetTokens(tokens token.Tokens) Builder {
	dynatraceClientBuilder.tokens = tokens

	return dynatraceClientBuilder
}

func (dynatraceClientBuilder builder) getTokens() token.Tokens {
	if dynatraceClientBuilder.tokens == nil {
		dynatraceClientBuilder.tokens = token.Tokens{}
	}

	return dynatraceClientBuilder.tokens
}

// Build creates a new Dynatrace client using the settings configured on the given instance.
func (dynatraceClientBuilder builder) Build(ctx context.Context) (dtclient.Client, error) {
	namespace := dynatraceClientBuilder.dk.Namespace
	apiReader := dynatraceClientBuilder.apiReader

	opts := newOptions(ctx)
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
