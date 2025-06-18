package edgeconnect

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OAuth struct {
	ClientID     string
	ClientSecret string
	Resource     string
}

func (ec *EdgeConnect) ClientSecretName() string {
	return ec.Name + "-client"
}

func (ec *EdgeConnect) GetOAuthClientFromSecret(ctx context.Context, kubeReader client.Reader, secretName string) (OAuth, error) {
	var clientSecret corev1.Secret

	oAuth := OAuth{}

	err := kubeReader.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ec.Namespace}, &clientSecret)
	if err != nil {
		return oAuth, errors.WithMessage(err, fmt.Sprintf("failed to get clientSecret from %s secret", secretName))
	}

	oauthClientID, hasKey := clientSecret.Data[consts.KeyEdgeConnectOauthClientID]
	if !hasKey {
		return oAuth, errors.Errorf("missing token %s in client secret %s", consts.KeyEdgeConnectOauthClientID, secretName)
	}

	oAuth.ClientID = string(oauthClientID)

	oauthClientSecret, hasKey := clientSecret.Data[consts.KeyEdgeConnectOauthClientSecret]
	if !hasKey {
		return oAuth, errors.Errorf("missing token %s in client secret %s", consts.KeyEdgeConnectOauthClientSecret, secretName)
	}

	oAuth.ClientSecret = string(oauthClientSecret)

	resource := ec.Spec.OAuth.Resource

	if ec.IsProvisionerModeEnabled() {
		resourceBytes, hasKey := clientSecret.Data[consts.KeyEdgeConnectOauthResource]
		if !hasKey {
			return oAuth, errors.Errorf("missing %s in %s", consts.KeyEdgeConnectOauthResource, secretName)
		}

		resource = string(resourceBytes)
	}

	oAuth.Resource = resource

	return oAuth, nil
}
