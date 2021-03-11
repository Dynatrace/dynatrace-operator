// +build e2e

package e2e

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createTokenSecret(clt client.Client, namespace string) error {
	apiToken, paasToken := GetTokensFromEnv()
	if apiToken == "" {
		return errors.New(fmt.Sprintf("variable %s must be set", keyAPIToken))
	}
	if paasToken == "" {
		return errors.New(fmt.Sprintf("variable %s must be set", keyPAASToken))
	}

	tokenSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TokenSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"apiToken":  []byte(apiToken),
			"paasToken": []byte(paasToken),
		},
	}
	err := clt.Create(context.TODO(), &tokenSecret)
	return errors.WithStack(err)
}

func GetTokensFromEnv() (string, string) {
	apiToken := os.Getenv(keyAPIToken)
	paasToken := os.Getenv(keyPAASToken)

	return apiToken, paasToken
}
