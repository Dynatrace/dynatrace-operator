package secret

import (
	"context"
	"reflect"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/query"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) query.Generic[*corev1.Secret, *corev1.SecretList] {
	return query.Generic[*corev1.Secret, *corev1.SecretList]{
		Target:     &corev1.Secret{},
		ListTarget: &corev1.SecretList{},
		ToList: func(sl *corev1.SecretList) []*corev1.Secret {
			out := []*corev1.Secret{}
			for _, s := range sl.Items {
				out = append(out, &s)
			}

			return out
		},
		IsEqual: AreSecretsEqual,

		KubeClient: kubeClient,
		KubeReader: kubeReader,
		Log:        log,
	}
}

func AreSecretsEqual(secret *corev1.Secret, other *corev1.Secret) bool {
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

func GetDataFromSecretName(ctx context.Context, apiReader client.Reader, namespacedName types.NamespacedName, dataKey string, log logd.Logger) (string, error) {
	query := Query(nil, apiReader, log)

	secret, err := query.Get(ctx, namespacedName)
	if err != nil {
		return "", errors.WithStack(err)
	}

	value, err := ExtractToken(secret, dataKey)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return value, nil
}

type secretBuilderData = corev1.Secret
type secretBuilderModifier = builder.Modifier[secretBuilderData]

func Create(owner metav1.Object, mods ...secretBuilderModifier) (*corev1.Secret, error) {
	builderOfSecret := builder.NewBuilder(corev1.Secret{})
	secret, err := builderOfSecret.AddModifier(mods...).AddModifier(newSecretOwnerModifier(owner)).Build()

	return &secret, err
}

func newSecretOwnerModifier(owner metav1.Object) secretOwnerModifier {
	return secretOwnerModifier{
		owner: owner,
	}
}

type secretOwnerModifier struct {
	owner metav1.Object
}

func (mod secretOwnerModifier) Enabled() bool {
	return true
}

func (mod secretOwnerModifier) Modify(secret *corev1.Secret) error {
	if err := controllerutil.SetControllerReference(mod.owner, secret, scheme.Scheme); err != nil {
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
