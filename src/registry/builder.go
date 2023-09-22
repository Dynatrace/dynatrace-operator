package registry

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientBuilder interface {
	SetContext(context.Context) ClientBuilder
	SetApiReader(client.Reader) ClientBuilder
	SetDynakube(*dynatracev1beta1.DynaKube) ClientBuilder
	Build() (*Client, error)
}

type builder struct {
	ctx       context.Context
	apiReader client.Reader
	dynakube  *dynatracev1beta1.DynaKube
}

func NewClientBuilder() ClientBuilder {
	return builder{}
}

func (builder builder) SetContext(ctx context.Context) ClientBuilder {
	builder.ctx = ctx
	return builder
}

func (builder builder) SetApiReader(apiReader client.Reader) ClientBuilder {
	builder.apiReader = apiReader
	return builder
}

func (builder builder) SetDynakube(dynakube *dynatracev1beta1.DynaKube) ClientBuilder {
	builder.dynakube = dynakube
	return builder
}

func (builder builder) Build() (*Client, error) {
	return NewClient(builder.ctx, builder.apiReader, builder.dynakube)
}
