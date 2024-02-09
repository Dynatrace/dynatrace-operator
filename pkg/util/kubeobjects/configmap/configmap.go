package configmap

import (
	"context"
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/query"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
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

func NewQuery(ctx context.Context, kubeClient client.Client, kubeReader client.Reader, log logger.DtLogger) Query {
	return Query{
		query.New(ctx, kubeClient, kubeReader, log),
	}
}

func (query Query) Get(objectKey client.ObjectKey) (corev1.ConfigMap, error) {
	var configMap corev1.ConfigMap
	err := query.KubeReader.Get(query.Ctx, objectKey, &configMap)

	return configMap, errors.WithStack(err)
}

func (query Query) Create(configMap corev1.ConfigMap) error {
	query.Log.Info("creating configMap", "name", configMap.Name, "namespace", configMap.Namespace)

	return errors.WithStack(query.KubeClient.Create(query.Ctx, &configMap))
}

func (query Query) Update(configMap corev1.ConfigMap) error {
	query.Log.Info("updating configMap", "name", configMap.Name, "namespace", configMap.Namespace)

	return errors.WithStack(query.KubeClient.Update(query.Ctx, &configMap))
}

func (query Query) Delete(configMap corev1.ConfigMap) error {
	query.Log.Info("removing configMap", "name", configMap.Name, "namespace", configMap.Namespace)

	err := query.KubeClient.Delete(query.Ctx, &configMap)
	if k8serrors.IsNotFound(err) {
		return nil
	}

	return errors.WithStack(err)
}

func (query Query) CreateOrUpdate(configMap corev1.ConfigMap) error {
	currentConfigMap, err := query.Get(types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = query.Create(configMap)
			if err != nil {
				return errors.WithStack(err)
			}

			return nil
		}

		return errors.WithStack(err)
	}

	if AreConfigMapsEqual(configMap, currentConfigMap) {
		query.Log.Info("configMap unchanged", "name", configMap.Name, "namespace", configMap.Namespace)

		return nil
	}

	err = query.Update(configMap)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func AreConfigMapsEqual(configMap corev1.ConfigMap, other corev1.ConfigMap) bool {
	return reflect.DeepEqual(configMap.Data, other.Data) && reflect.DeepEqual(configMap.Labels, other.Labels) && reflect.DeepEqual(configMap.OwnerReferences, other.OwnerReferences)
}

type configMapData = corev1.ConfigMap
type configMapModifier = builder.Modifier[configMapData]

func CreateConfigMap(scheme *runtime.Scheme, owner metav1.Object, mods ...configMapModifier) (*corev1.ConfigMap, error) {
	builderOfSecret := builder.NewBuilder(corev1.ConfigMap{})
	secret, err := builderOfSecret.AddModifier(mods...).AddModifier(newConfigMapOwnerModifier(scheme, owner)).Build()

	return &secret, err
}

func newConfigMapOwnerModifier(scheme *runtime.Scheme, owner metav1.Object) configMapOwnerModifier {
	return configMapOwnerModifier{
		scheme: scheme,
		owner:  owner,
	}
}

type configMapOwnerModifier struct {
	scheme *runtime.Scheme
	owner  metav1.Object
}

func (mod configMapOwnerModifier) Enabled() bool {
	return true
}

func (mod configMapOwnerModifier) Modify(secret *corev1.ConfigMap) error {
	if err := controllerutil.SetControllerReference(mod.owner, secret, mod.scheme); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func NewModifier(name string) Modifier {
	return Modifier{
		name: name,
	}
}

type Modifier struct {
	name string
}

func (mod Modifier) Enabled() bool {
	return true
}

func (mod Modifier) Modify(configMap *corev1.ConfigMap) error {
	configMap.Name = mod.name

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

func (mod NamespaceModifier) Modify(configMap *corev1.ConfigMap) error {
	configMap.Namespace = mod.namespaceName

	return nil
}

func NewConfigMapDataModifier(data map[string]string) DataModifier {
	return DataModifier{
		data: data,
	}
}

type DataModifier struct {
	data map[string]string
}

func (mod DataModifier) Enabled() bool {
	return true
}

func (mod DataModifier) Modify(configMap *corev1.ConfigMap) error {
	configMap.Data = mod.data

	return nil
}
