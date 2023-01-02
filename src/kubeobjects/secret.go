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

type SecretQuery struct {
	kubeQuery
}

func NewSecretQuery(ctx context.Context, kubeClient client.Client, kubeReader client.Reader, log logr.Logger) SecretQuery {
	return SecretQuery{
		newKubeQuery(ctx, kubeClient, kubeReader, log),
	}
}

func (query SecretQuery) Get(objectKey client.ObjectKey) (corev1.Secret, error) {
	var secret corev1.Secret
	err := query.kubeReader.Get(query.ctx, objectKey, &secret)

	return secret, errors.WithStack(err)
}

func (query SecretQuery) Create(secret corev1.Secret) error {
	query.log.Info("creating secret", "name", secret.Name, "namespace", secret.Namespace)

	return errors.WithStack(query.kubeClient.Create(query.ctx, &secret))
}

func (query SecretQuery) Update(secret corev1.Secret) error {
	query.log.Info("updating secret", "name", secret.Name, "namespace", secret.Namespace)

	return errors.WithStack(query.kubeClient.Update(query.ctx, &secret))
}

func (query SecretQuery) GetAllFromNamespaces(secretName string) ([]corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	listOps := []client.ListOption{
		client.MatchingFields{
			"metadata.name": secretName, // todo delete comment - config.AgentInitSecretName,
		},
	}
	err := query.kubeReader.List(query.ctx, secretList, listOps...)

	if client.IgnoreNotFound(err) != nil {
		return nil, errors.WithStack(err)
	}
	return secretList.Items, err
}

func (query SecretQuery) CreateOrUpdate(secret corev1.Secret) error {
	currentSecret, err := query.Get(types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = query.Create(secret)
			if err != nil {
				return errors.WithStack(err)
			}
			return nil
		}
		return errors.WithStack(err)
	}

	if AreSecretsEqual(secret, currentSecret) {
		return nil
	}

	err = query.Update(secret)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (query SecretQuery) CreateOrUpdateForNamespacesList(newSecret corev1.Secret, namespaces []corev1.Namespace) (int, error) {
	secretList, err := query.GetAllFromNamespaces(newSecret.Name)
	if err != nil {
		return 0, err
	}

	namespacesContainingSecret := make(map[string]corev1.Secret)
	for _, secret := range secretList {
		namespacesContainingSecret[secret.Namespace] = secret
	}

	updateAndCreationCount := 0

	for _, namespace := range namespaces {
		newSecret.Namespace = namespace.Name

		if oldSecret, ok := namespacesContainingSecret[namespace.Name]; ok {
			if !AreSecretsEqual(oldSecret, newSecret) {
				err = query.Update(newSecret)
				if err != nil {
					return updateAndCreationCount, err
				}
				updateAndCreationCount++
			}
		} else {
			err = query.Create(newSecret)
			if err != nil {
				return updateAndCreationCount, err
			}
			updateAndCreationCount++
		}
	}

	return updateAndCreationCount, nil
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

	// Errors would have been caught by verifySecret
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

func GetDataFromSecretName(apiReader client.Reader, namespacedName types.NamespacedName, dataKey string, log logr.Logger) (string, error) {
	query := NewSecretQuery(context.TODO(), nil, apiReader, log)
	secret, err := query.Get(namespacedName)
	if err != nil {
		return "", errors.WithStack(err)
	}

	value, err := ExtractToken(&secret, dataKey)
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

func IsSecretDataEqual(currentSecret *corev1.Secret, desired map[string][]byte) bool {
	return reflect.DeepEqual(desired, currentSecret.Data)
}
