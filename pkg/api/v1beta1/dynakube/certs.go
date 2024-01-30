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
	"encoding/pem"
	"fmt"
	"strings"

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

		blocks, err := pkcs12.ToPEM(tlsSecret.Data[TlsP12Key], string(tlsSecret.Data[TlsP12PasswordKey]))
		if err != nil {
			return "", err
		}

		certs := bytes.NewBufferString("")
		for _, block := range blocks {
			if strings.Contains(block.Type, "PRIVATE KEY") {
				continue
			}

			// everything else should be certificate chain

			err = pem.Encode(certs, &pem.Block{Type: "CERTIFICATE", Bytes: block.Bytes})
			if err != nil {
				return "", err
			}

			certs.WriteString("\n")
		}

		return certs.String(), nil
	}

	return "", nil
}
