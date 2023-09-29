package registry

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientBuilder interface {
	SetContext(context.Context) ClientBuilder
	SetApiReader(client.Reader) ClientBuilder
	SetKeyChainSecret(*corev1.Secret) ClientBuilder
	SetProxy(string) ClientBuilder
	SetTrustedCAs([]byte) ClientBuilder
	Build() (ImageGetter, error)
}

type builder struct {
	ctx            context.Context
	apiReader      client.Reader
	keyChainSecret *corev1.Secret
	proxy          string
	trustedCAs     []byte
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

func (builder builder) SetKeyChainSecret(keyChainSecret *corev1.Secret) ClientBuilder {
	builder.keyChainSecret = keyChainSecret
	return builder
}

func (builder builder) SetProxy(proxy string) ClientBuilder {
	builder.proxy = proxy
	return builder
}

func (builder builder) SetTrustedCAs(trustedCAs []byte) ClientBuilder {
	builder.trustedCAs = trustedCAs
	return builder
}

func (builder builder) Build() (ImageGetter, error) {
	return NewClient(builder.ctx, builder.apiReader, builder.keyChainSecret, builder.proxy, builder.trustedCAs)
}
