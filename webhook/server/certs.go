package server

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/certificates/validation"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateCertificateForWebhook(apiReader client.Reader, fs afero.Fs, certDir string, ns string) (bool, error) {
	return UpdateCertificatesForSecret(apiReader, fs, certDir, ns, webhook.SecretCertsName)
}

func UpdateCertificateForValidation(apiReader client.Reader, fs afero.Fs, certDir string, ns string) (bool, error) {
	return UpdateCertificatesForSecret(apiReader, fs, certDir, ns, validation.SecretCertsName)
}

func UpdateCertificatesForSecret(apiReader client.Reader, fs afero.Fs, certDir string, ns string, secretName string) (bool, error) {
	var secret corev1.Secret

	err := apiReader.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: ns}, &secret)
	if err != nil {
		return false, err
	}

	if _, err := fs.Stat(certDir); os.IsNotExist(err) {
		err = fs.MkdirAll(certDir, 0755)
		if err != nil {
			return false, fmt.Errorf("could not create cert directory: %s", err)
		}
	}

	for _, filename := range []string{"tls.crt", "tls.key"} {
		f := filepath.Join(certDir, filename)

		data, err := afero.ReadFile(fs, f)

		if os.IsNotExist(err) || !bytes.Equal(data, secret.Data[filename]) {
			if err := afero.WriteFile(fs, f, secret.Data[filename], 0666); err != nil {
				return false, err
			}
		} else {
			return false, err
		}
	}

	return true, nil
}
