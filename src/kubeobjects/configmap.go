package kubeobjects

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func NewConfigMap(name string, namespace string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func IsConfigMapDataEqual(currentConfigMap *corev1.ConfigMap, desired map[string]string) bool {
	return reflect.DeepEqual(desired, currentConfigMap.Data)
}
