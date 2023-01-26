package kubeobjects

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	return query.create(secret)
}

func (query SecretQuery) Update(secret corev1.Secret) error {
	query.log.Info("updating secret", "name", secret.Name, "namespace", secret.Namespace)

	return query.update(secret)
}

func (query SecretQuery) create(secret corev1.Secret) error {
	return errors.WithStack(query.kubeClient.Create(query.ctx, &secret))
}

func (query SecretQuery) update(secret corev1.Secret) error {
	return errors.WithStack(query.kubeClient.Update(query.ctx, &secret))
}

func (query SecretQuery) GetAllFromNamespaces(secretName string) ([]corev1.Secret, error) {
	query.log.Info("querying secret from all namespaces", "name", secretName)

	secretList := &corev1.SecretList{}
	listOps := []client.ListOption{
		client.MatchingFields{
			"metadata.name": secretName,
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

func (query SecretQuery) CreateOrUpdateForNamespacesList(newSecret corev1.Secret, namespaces []corev1.Namespace) error {
	secretList, err := query.GetAllFromNamespaces(newSecret.Name)
	if err != nil {
		return err
	}

	query.log.Info("reconciling secret for multiple namespaces",
		"name", newSecret.Name, "len(namespaces)", len(namespaces))

	namespacesContainingSecret := make(map[string]corev1.Secret, len(secretList))
	for _, secret := range secretList {
		namespacesContainingSecret[secret.Namespace] = secret
	}

	updateCount := 0
	creationCount := 0

	for _, namespace := range namespaces {
		newSecret.Namespace = namespace.Name

		if oldSecret, ok := namespacesContainingSecret[namespace.Name]; ok {
			if !AreSecretsEqual(oldSecret, newSecret) {
				err = query.update(newSecret)
				if err != nil {
					return err
				}
				updateCount++
			}
		} else {
			err = query.create(newSecret)
			if err != nil {
				return err
			}
			creationCount++
		}
	}

	query.log.Info("reconciled secret for multiple namespaces",
		"name", newSecret.Name, "creationCount", creationCount, "updateCount", updateCount)

	return nil
}

func AreSecretsEqual(secret corev1.Secret, other corev1.Secret) bool {
	return reflect.DeepEqual(secret.Data, other.Data) && reflect.DeepEqual(secret.Labels, other.Labels) && reflect.DeepEqual(secret.OwnerReferences, other.OwnerReferences)
}

type Tokens struct {
	ApiToken  string
	PaasToken string
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

type SecretBuilder struct {
	scheme     *runtime.Scheme
	owner      metav1.Object
	secretType corev1.SecretType
	labels     map[string]string
}

func NewSecretBuilder(scheme *runtime.Scheme, owner metav1.Object) *SecretBuilder {
	return &SecretBuilder{
		scheme: scheme,
		owner:  owner,
	}
}

func (secretBuilder *SecretBuilder) WithType(secretType corev1.SecretType) *SecretBuilder {
	secretBuilder.secretType = secretType
	return secretBuilder
}

func (secretBuilder *SecretBuilder) WithLables(labels map[string]string) *SecretBuilder {
	if secretBuilder.labels == nil {
		secretBuilder.labels = map[string]string{}
	}
	for k, v := range labels {
		secretBuilder.labels[k] = v
	}
	return secretBuilder
}

func (secretBuilder *SecretBuilder) Build(name string, namespace string, data map[string][]byte) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    secretBuilder.labels,
		},
		Type: secretBuilder.secretType,
		Data: data,
	}

	if err := controllerutil.SetControllerReference(secretBuilder.owner, secret, secretBuilder.scheme); err != nil {
		return nil, errors.WithStack(err)
	}
	return secret, nil
}
