package secret

import (
	"context"
	"reflect"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/query"
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

type Query struct {
	query.KubeQuery
}

func NewQuery(ctx context.Context, kubeClient client.Client, kubeReader client.Reader, log logr.Logger) Query {
	return Query{
		query.New(ctx, kubeClient, kubeReader, log),
	}
}

func (query Query) Get(objectKey client.ObjectKey) (corev1.Secret, error) {
	var secret corev1.Secret
	err := query.KubeReader.Get(query.Ctx, objectKey, &secret)

	return secret, errors.WithStack(err)
}

func (query Query) Create(secret corev1.Secret) error {
	query.Log.Info("creating secret", "name", secret.Name, "namespace", secret.Namespace)

	return query.create(secret)
}

func (query Query) Update(secret corev1.Secret) error {
	query.Log.Info("updating secret", "name", secret.Name, "namespace", secret.Namespace)

	return query.update(secret)
}

func (query Query) create(secret corev1.Secret) error {
	return errors.WithStack(query.KubeClient.Create(query.Ctx, &secret))
}

func (query Query) update(secret corev1.Secret) error {
	return errors.WithStack(query.KubeClient.Update(query.Ctx, &secret))
}

func (query Query) GetAllFromNamespaces(secretName string) ([]corev1.Secret, error) {
	query.Log.Info("querying secret from all namespaces", "name", secretName)

	secretList := &corev1.SecretList{}
	listOps := []client.ListOption{
		client.MatchingFields{
			"metadata.name": secretName,
		},
	}
	err := query.KubeReader.List(query.Ctx, secretList, listOps...)

	if client.IgnoreNotFound(err) != nil {
		return nil, errors.WithStack(err)
	}
	return secretList.Items, err
}

func (query Query) CreateOrUpdate(secret corev1.Secret) error {
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

func (query Query) CreateOrUpdateForNamespacesList(newSecret corev1.Secret, namespaces []corev1.Namespace) error {
	secretList, err := query.GetAllFromNamespaces(newSecret.Name)
	if err != nil {
		return err
	}

	query.Log.Info("reconciling secret for multiple namespaces",
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

	query.Log.Info("reconciled secret for multiple namespaces",
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
		err := errors.Errorf("missing token %s", key)
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func GetDataFromSecretName(apiReader client.Reader, namespacedName types.NamespacedName, dataKey string, log logr.Logger) (string, error) {
	query := NewQuery(context.TODO(), nil, apiReader, log)
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

type secretBuilderData = corev1.Secret
type secretBuilderModifier = builder.Modifier[secretBuilderData]

func CreateSecret(scheme *runtime.Scheme, owner metav1.Object, mods ...secretBuilderModifier) (*corev1.Secret, error) {
	builderOfSecret := builder.NewBuilder(corev1.Secret{})
	secret, err := builderOfSecret.AddModifier(mods...).AddModifier(newSecretOwnerModifier(scheme, owner)).Build()
	return &secret, err
}

func newSecretOwnerModifier(scheme *runtime.Scheme, owner metav1.Object) secretOwnerModifier {
	return secretOwnerModifier{
		scheme: scheme,
		owner:  owner,
	}
}

type secretOwnerModifier struct {
	scheme *runtime.Scheme
	owner  metav1.Object
}

func (mod secretOwnerModifier) Enabled() bool {
	return true
}

func (mod secretOwnerModifier) Modify(secret *corev1.Secret) error {
	if err := controllerutil.SetControllerReference(mod.owner, secret, mod.scheme); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func NewNameModifier(name string) NameModifier {
	return NameModifier{
		name: name,
	}
}

type NameModifier struct {
	name string
}

func (mod NameModifier) Enabled() bool {
	return true
}

func (mod NameModifier) Modify(secret *corev1.Secret) error {
	secret.Name = mod.name
	return nil
}

func NewNamespaceModifier(namespaceName string) NamespaceModifier {
	return NamespaceModifier{
		namespaceName: namespaceName,
	}
}

type NamespaceModifier struct {
	namespaceName string
}

func (mod NamespaceModifier) Enabled() bool {
	return true
}

func (mod NamespaceModifier) Modify(secret *corev1.Secret) error {
	secret.Namespace = mod.namespaceName
	return nil
}

func NewDataModifier(data map[string][]byte) DataModifier {
	return DataModifier{
		data: data,
	}
}

type DataModifier struct {
	data map[string][]byte
}

func (mod DataModifier) Enabled() bool {
	return true
}

func (mod DataModifier) Modify(secret *corev1.Secret) error {
	secret.Data = mod.data
	return nil
}

func NewTypeModifier(secretType corev1.SecretType) TypeModifier {
	return TypeModifier{
		secretType: secretType,
	}
}

type TypeModifier struct {
	secretType corev1.SecretType
}

func (mod TypeModifier) Enabled() bool {
	return true
}

func (mod TypeModifier) Modify(secret *corev1.Secret) error {
	secret.Type = mod.secretType
	return nil
}

func NewLabelsModifier(labels map[string]string) LabelsModifier {
	return LabelsModifier{
		labels: labels,
	}
}

type LabelsModifier struct {
	labels map[string]string
}

func (mod LabelsModifier) Enabled() bool {
	return true
}

func (mod LabelsModifier) Modify(secret *corev1.Secret) error {
	secret.Labels = mod.labels
	return nil
}
