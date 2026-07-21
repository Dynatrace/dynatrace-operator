// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TrustedCAKey  = "certs"
	ServerCertKey = "server.crt"
	TLSCertKey    = "tls.crt"
)

func (dk *DynaKube) TrustedCAs(ctx context.Context, kubeReader client.Reader) ([]byte, error) {
	configName := dk.Spec.TrustedCAs
	if configName != "" {
		var caConfigMap corev1.ConfigMap

		err := kubeReader.Get(ctx, client.ObjectKey{Name: configName, Namespace: dk.Namespace}, &caConfigMap)
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("failed to get trustedCa from %s configmap", configName))
		}

		return []byte(caConfigMap.Data[TrustedCAKey]), nil
	}

	return nil, nil
}

func (dk *DynaKube) ActiveGateTLSCert(ctx context.Context, kubeReader client.Reader) ([]byte, error) {
	if dk.ActiveGate().HasCaCert() {
		secretName := dk.Spec.ActiveGate.GetTLSSecretName()

		var tlsSecret corev1.Secret

		err := kubeReader.Get(ctx, client.ObjectKey{Name: secretName, Namespace: dk.Namespace}, &tlsSecret)
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("failed to get activeGate tlsCert from %s secret", secretName))
		}

		// first check if the tls.crt key is available
		if tlsCertKey, ok := tlsSecret.Data[TLSCertKey]; ok {
			return tlsCertKey, nil
		}

		// use server.crt as fallback for older secrets
		if tlsCertKey, ok := tlsSecret.Data[ServerCertKey]; ok {
			return tlsCertKey, nil
		}
	}

	return nil, nil
}
