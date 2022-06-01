package kubeobjects

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateSecretIfNotExists creates a secret in case it does not exist or updates it if there are changes
func CreateOrUpdateSecretIfNotExists(c client.Client, r client.Reader, secret *corev1.Secret, log logr.Logger) (bool, error) {
	var cfg corev1.Secret
	err := r.Get(context.TODO(), client.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, &cfg)
	if k8serrors.IsNotFound(err) {
		log.Info("creating secret", "namespace", secret.Namespace, "secret", secret.Name)
		if err := c.Create(context.TODO(), secret); err != nil {
			return false, errors.Wrapf(err, "failed to create secret %s", secret.Name)
		}
		return true, nil
	}

	if err != nil {
		return false, errors.Wrapf(err, "failed to query for secret %s", secret.Name)
	}
	var updated bool
	if !reflect.DeepEqual(secret.Data, cfg.Data) {
		updated = true
		cfg.Data = secret.Data
	}

	if !reflect.DeepEqual(secret.Labels, cfg.Labels) {
		updated = true
		cfg.Labels = secret.Labels
	}

	if updated {
		if err := c.Update(context.TODO(), &cfg); err != nil {
			log.Info("updating secret", "namespace", secret.Namespace, "secret", secret.Name)
			return false, errors.Wrapf(err, "failed to update secret %s", secret.Name)
		}
		return true, nil
	}

	return false, nil
}

type Tokens struct {
	ApiToken  string
	PaasToken string
}

func NewTokens(secret *corev1.Secret) (*Tokens, error) {
	if secret == nil {
		return nil, fmt.Errorf("could not parse tokens: secret is nil")
	}

	var apiToken string
	var paasToken string
	var err error

	if err = verifySecret(secret); err != nil {
		return nil, errors.WithStack(err)
	}

	//Errors would have been caught by verifySecret
	apiToken, _ = ExtractToken(secret, dtclient.DynatraceApiToken)
	paasToken, err = ExtractToken(secret, dtclient.DynatracePaasToken)
	if err != nil {
		paasToken = apiToken
	}

	return &Tokens{
		ApiToken:  apiToken,
		PaasToken: paasToken,
	}, nil
}

func verifySecret(secret *corev1.Secret) error {
	for _, token := range []string{
		dtclient.DynatraceApiToken} {
		_, err := ExtractToken(secret, token)
		if err != nil {
			return errors.Errorf("invalid secret %s, %s", secret.Name, err)
		}
	}

	return nil
}

func ExtractToken(secret *corev1.Secret, key string) (string, error) {
	value, hasKey := secret.Data[key]
	if !hasKey {
		err := fmt.Errorf("missing token %s", key)
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func GetSecret(ctx context.Context, apiReader client.Reader, namespacedName types.NamespacedName) (*corev1.Secret, error) {
	var secret corev1.Secret
	err := apiReader.Get(ctx, namespacedName, &secret)
	return &secret, errors.WithStack(err)
}

func GetDataFromSecretName(apiReader client.Reader, namespacedName types.NamespacedName, dataKey string) (string, error) {
	secret, err := GetSecret(context.TODO(), apiReader, namespacedName)
	if err != nil {
		return "", errors.WithStack(err)
	}

	value, err := ExtractToken(secret, dataKey)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return value, nil
}

func NewSecret(name string, namespace string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func IsSecretEqual(currentSecret *corev1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}
