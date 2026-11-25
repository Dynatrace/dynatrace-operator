// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package tls

import (
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	corev1 "k8s.io/api/core/v1"
)

func CreateTestdataTLSSecret(namespace string, name string, keyFile string, crtFile string) (*corev1.Secret, error) {
	tlsCrt, err := os.ReadFile(filepath.Join(project.TestDataDir(), crtFile))
	if err != nil {
		return &corev1.Secret{}, err
	}

	tlsKey, err := os.ReadFile(filepath.Join(project.TestDataDir(), keyFile))
	if err != nil {
		return &corev1.Secret{}, err
	}

	tlsSecret := k8ssecret.New(name, namespace,
		map[string][]byte{
			corev1.TLSCertKey:       tlsCrt,
			corev1.TLSPrivateKeyKey: tlsKey,
		})
	tlsSecret.Type = corev1.SecretTypeTLS

	return tlsSecret, nil
}
