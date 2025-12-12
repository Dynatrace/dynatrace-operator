//go:build e2e

package tls

import (
	"os"
	"path/filepath"

	operatorconsts "github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	corev1 "k8s.io/api/core/v1"
)

func CreateTestdataTLSSecret(namespace string, name string, keyFile string, crtFile string) (corev1.Secret, error) {
	tlsCrt, err := os.ReadFile(filepath.Join(project.TestDataDir(), crtFile))
	if err != nil {
		return corev1.Secret{}, err
	}

	tlsKey, err := os.ReadFile(filepath.Join(project.TestDataDir(), keyFile))
	if err != nil {
		return corev1.Secret{}, err
	}

	tlsSecret := secret.New(name, namespace,
		map[string][]byte{
			operatorconsts.TLSCrtDataName: tlsCrt,
			operatorconsts.TLSKeyDataName: tlsKey,
		})
	tlsSecret.Type = corev1.SecretTypeTLS

	return tlsSecret, nil
}
