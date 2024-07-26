package edgeconnect

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ProxyAuthUserKey     = "user"
	ProxyAuthPasswordKey = "password"
)

func (ec *EdgeConnect) ProxyAuth(ctx context.Context, kubeReader client.Reader) (string, string, error) {
	if ec.Spec.Proxy == nil {
		return "", "", nil
	}

	if ec.Spec.Proxy.AuthRef != "" {
		return ec.proxyAuthFromUserSecret(ctx, kubeReader)
	}

	return "", "", nil
}

func (ec *EdgeConnect) proxyAuthFromUserSecret(ctx context.Context, kubeReader client.Reader) (string, string, error) {
	secretName := ec.Spec.Proxy.AuthRef

	var proxySecret corev1.Secret

	err := kubeReader.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ec.Namespace}, &proxySecret)
	if err != nil {
		return "", "", errors.WithMessage(err, fmt.Sprintf("failed to get proxy from %s secret", secretName))
	}

	user, hasKey := proxySecret.Data[ProxyAuthUserKey]
	if !hasKey {
		err := errors.Errorf("missing token %s in proxy secret %s", ProxyAuthUserKey, secretName)

		return "", "", err
	}

	password, hasKey := proxySecret.Data[ProxyAuthPasswordKey]
	if !hasKey {
		err := errors.Errorf("missing token %s in proxy secret %s", ProxyAuthPasswordKey, secretName)

		return "", "", err
	}

	return string(user), string(password), nil
}
