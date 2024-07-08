package configmap

import (
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/query"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) query.Generic[*corev1.ConfigMap, *corev1.ConfigMapList] {
	return query.Generic[*corev1.ConfigMap, *corev1.ConfigMapList]{
		Target:     &corev1.ConfigMap{},
		ListTarget: &corev1.ConfigMapList{},
		ToList: func(cml *corev1.ConfigMapList) []*corev1.ConfigMap {
			out := []*corev1.ConfigMap{}
			for _, cm := range cml.Items {
				out = append(out, &cm)
			}

			return out
		},
		IsEqual: AreConfigMapsEqual,

		KubeClient: kubeClient,
		KubeReader: kubeReader,
		Log:        log,
	}
}

func AreConfigMapsEqual(configMap *corev1.ConfigMap, other *corev1.ConfigMap) bool {
	return reflect.DeepEqual(configMap.Data, other.Data) && reflect.DeepEqual(configMap.Labels, other.Labels) && reflect.DeepEqual(configMap.OwnerReferences, other.OwnerReferences)
}

type configMapData = corev1.ConfigMap
type configMapModifier = builder.Modifier[configMapData]

func CreateConfigMap(owner metav1.Object, mods ...configMapModifier) (*corev1.ConfigMap, error) {
	builderOfSecret := builder.NewBuilder(corev1.ConfigMap{})
	secret, err := builderOfSecret.AddModifier(mods...).AddModifier(newConfigMapOwnerModifier(owner)).Build()

	return &secret, err
}

func newConfigMapOwnerModifier(owner metav1.Object) configMapOwnerModifier {
	return configMapOwnerModifier{
		owner: owner,
	}
}

type configMapOwnerModifier struct {
	owner metav1.Object
}

func (mod configMapOwnerModifier) Enabled() bool {
	return true
}

func (mod configMapOwnerModifier) Modify(secret *corev1.ConfigMap) error {
	if err := controllerutil.SetControllerReference(mod.owner, secret, scheme.Scheme); err != nil {
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
