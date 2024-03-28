package edgeconnect

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (ec *EdgeConnect) ClientSecretName() string {
	return ec.Name + "-client"
}

func (ec *EdgeConnect) GetOAuthClientFromSecret(ctx context.Context, kubeReader client.Reader, secretName string) (string, string, error) {
	var clientSecret corev1.Secret

	err := kubeReader.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ec.Namespace}, &clientSecret)
	if err != nil {
		return "", "", errors.WithMessage(err, fmt.Sprintf("failed to get clientSecret from %s secret", secretName))
	}

	oauthClientId, hasKey := clientSecret.Data[consts.KeyEdgeConnectOauthClientID]
	if !hasKey {
		return "", "", errors.Errorf("missing token %s in client secret %s", consts.KeyEdgeConnectOauthClientID, secretName)
	}

	oauthClientSecret, hasKey := clientSecret.Data[consts.KeyEdgeConnectOauthClientSecret]
	if !hasKey {
		return "", "", errors.Errorf("missing token %s in client secret %s", consts.KeyEdgeConnectOauthClientSecret, secretName)
	}

	return string(oauthClientId), string(oauthClientSecret), nil
}
