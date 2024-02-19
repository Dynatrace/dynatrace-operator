//go:build e2e

package edgeconnect

import (
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultName      = "edgeconnect"
	defaultNamespace = "dynatrace"
)

type Option func(edgeconnect *edgeconnectv1alpha1.EdgeConnect)

func New(opts ...Option) *edgeconnectv1alpha1.EdgeConnect {
	edgeconnect := &edgeconnectv1alpha1.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultName,
			Namespace: defaultNamespace,
		},
		Spec:   edgeconnectv1alpha1.EdgeConnectSpec{},
		Status: edgeconnectv1alpha1.EdgeConnectStatus{},
	}
	for _, opt := range opts {
		opt(edgeconnect)
	}

	return edgeconnect
}

func WithName(name string) Option {
	return func(edgeconnect *edgeconnectv1alpha1.EdgeConnect) {
		edgeconnect.ObjectMeta.Name = name
	}
}

func WithApiServer(apiURL string) Option {
	return func(edgeconnect *edgeconnectv1alpha1.EdgeConnect) {
		edgeconnect.Spec.ApiServer = apiURL
	}
}

func WithOAuthClientSecret(clientSecretName string) Option {
	return func(edgeconnect *edgeconnectv1alpha1.EdgeConnect) {
		edgeconnect.Spec.OAuth.ClientSecret = clientSecretName
	}
}

func WithOAuthResource(resource string) Option {
	return func(edgeconnect *edgeconnectv1alpha1.EdgeConnect) {
		edgeconnect.Spec.OAuth.Resource = resource
	}
}

func WithOAuthEndpoint(endpoint string) Option {
	return func(edgeconnect *edgeconnectv1alpha1.EdgeConnect) {
		edgeconnect.Spec.OAuth.Endpoint = endpoint
	}
}
