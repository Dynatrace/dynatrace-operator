/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dynakube

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"software.sslmate.com/src/go-pkcs12"
)

const (
	TrustedCAKey      = "certs"
	TlsCertKey        = "server.crt"
	TlsP12Key         = "server.p12"
	TlsP12PasswordKey = "password"
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

func (dk *DynaKube) ActiveGateTlsCert(ctx context.Context, kubeReader client.Reader) (string, error) {
	if dk.HasActiveGateCaCert() {
		secretName := dk.Spec.ActiveGate.TlsSecretName

		var tlsSecret corev1.Secret

		err := kubeReader.Get(ctx, client.ObjectKey{Name: secretName, Namespace: dk.Namespace}, &tlsSecret)
		if err != nil {
			return "", errors.WithMessage(err, fmt.Sprintf("failed to get activeGate tlsCert from %s secret", secretName))
		}

		// simply return server.crt certificates if provided by user
		if tlsCertKey, ok := tlsSecret.Data[TlsCertKey]; ok {
			return string(tlsCertKey), nil
		}

		// otherwise extract AG certificate (+ chain of root certificates) from server.p12 file (which is directly consumed by AG)

		// ignore privateKey
		_, certificate, caCerts, err := pkcs12.DecodeChain(tlsSecret.Data[TlsP12Key], string(tlsSecret.Data[TlsP12PasswordKey]))
		if err != nil {
			return "", err
		}

		cas := []*x509.Certificate{
			certificate,
		}
		cas = append(cas, caCerts...)

		certs := bytes.NewBufferString("")
		for _, ca := range cas {
			err = pem.Encode(certs, &pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})
			if err != nil {
				return "", err
			}

			certs.WriteString("\n")
		}

		return certs.String(), nil
	}

	return "", nil
}
