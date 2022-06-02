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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SecretQuery struct {
	complexKubeQuery
}

func NewSecretQuery(ctx context.Context, kubeClient client.Client, kubeReader client.Reader, log logr.Logger) SecretQuery {
	return SecretQuery{
		newComplexKubeQuery(ctx, kubeClient, kubeReader, log),
	}
}

func (query SecretQuery) Create(secret corev1.Secret) error {
	ctx := query.ctx
	kubeClient := query.kubeClient

	query.log.Info("creating secret", "name", secret.Name, "namespace", secret.Namespace)

	return errors.WithStack(kubeClient.Create(ctx, &secret))
}

func (query SecretQuery) Update(secret corev1.Secret) error {
	ctx := query.ctx
	kubeClient := query.kubeClient

	query.log.Info("updating secret", "name", secret.Name, "namespace", secret.Namespace)

	return errors.WithStack(kubeClient.Update(ctx, &secret))
}

func (query SecretQuery) CreateOrUpdate(secret corev1.Secret) error {
	err := query.Create(secret)

	if !k8serrors.IsAlreadyExists(err) {
		return errors.WithStack(err)
	}

	query.log.Info("secret already exists", "name", secret.Name, "namespace", secret.Namespace)

	return errors.WithStack(query.Update(secret))
}

func AreSecretsEqual(secret corev1.Secret, other corev1.Secret) bool {
	return reflect.DeepEqual(secret.Data, other.Data) && reflect.DeepEqual(secret.Labels, other.Labels)
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

func GetSecret(ctx context.Context, apiReader client.Reader, name string, namespace string) (*corev1.Secret, error) {
	var secret corev1.Secret
	err := apiReader.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, &secret)
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	return &secret, errors.WithStack(err)
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

func IsSecretDataEqual(currentSecret *corev1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}
