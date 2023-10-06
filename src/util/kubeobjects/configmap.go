package kubeobjects

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/src/util/builder"
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

type ConfigMapQuery struct {
	kubeQuery
}

func NewConfigMapQuery(ctx context.Context, kubeClient client.Client, kubeReader client.Reader, log logr.Logger) ConfigMapQuery {
	return ConfigMapQuery{
		newKubeQuery(ctx, kubeClient, kubeReader, log),
	}
}

func (query ConfigMapQuery) Get(objectKey client.ObjectKey) (corev1.ConfigMap, error) {
	var configMap corev1.ConfigMap
	err := query.kubeReader.Get(query.ctx, objectKey, &configMap)

	return configMap, errors.WithStack(err)
}

func (query ConfigMapQuery) Create(configMap corev1.ConfigMap) error {
	query.log.Info("creating configMap", "name", configMap.Name, "namespace", configMap.Namespace)

	return errors.WithStack(query.kubeClient.Create(query.ctx, &configMap))
}

func (query ConfigMapQuery) Update(configMap corev1.ConfigMap) error {
	query.log.Info("updating configMap", "name", configMap.Name, "namespace", configMap.Namespace)

	return errors.WithStack(query.kubeClient.Update(query.ctx, &configMap))
}

func (query ConfigMapQuery) Delete(configMap corev1.ConfigMap) error {
	query.log.Info("removing configMap", "name", configMap.Name, "namespace", configMap.Namespace)
	err := query.kubeClient.Delete(query.ctx, &configMap)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return errors.WithStack(err)
}

func (query ConfigMapQuery) CreateOrUpdate(configMap corev1.ConfigMap) error {
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
		query.log.Info("configMap unchanged", "name", configMap.Name, "namespace", configMap.Namespace)
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

func ExtractField(configMap *corev1.ConfigMap, key string) (string, error) {
	value, hasKey := configMap.Data[key]
	if !hasKey {
		err := fmt.Errorf("missing field %s", key)
		return "", err
	}

	return strings.TrimSpace(value), nil
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

func NewConfigMapNameModifier(name string) ConfigMapNameModifier {
	return ConfigMapNameModifier{
		name: name,
	}
}

type ConfigMapNameModifier struct {
	name string
}

func (mod ConfigMapNameModifier) Enabled() bool {
	return true
}

func (mod ConfigMapNameModifier) Modify(configMap *corev1.ConfigMap) error {
	configMap.Name = mod.name
	return nil
}

func NewConfigMapNamespaceModifier(namespaceName string) ConfigMapNamespaceModifier {
	return ConfigMapNamespaceModifier{
		namespaceName: namespaceName,
	}
}

type ConfigMapNamespaceModifier struct {
	namespaceName string
}

func (mod ConfigMapNamespaceModifier) Enabled() bool {
	return true
}

func (mod ConfigMapNamespaceModifier) Modify(configMap *corev1.ConfigMap) error {
	configMap.Namespace = mod.namespaceName
	return nil
}

func NewConfigMapDataModifier(data map[string]string) ConfigMapDataModifier {
	return ConfigMapDataModifier{
		data: data,
	}
}

type ConfigMapDataModifier struct {
	data map[string]string
}

func (mod ConfigMapDataModifier) Enabled() bool {
	return true
}

func (mod ConfigMapDataModifier) Modify(configMap *corev1.ConfigMap) error {
	configMap.Data = mod.data
	return nil
}
