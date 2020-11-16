package activegate

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/builder"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileActiveGate) manageCustomProperties(name string, kubernetesMonitoringSpec *dynatracev1alpha1.KubernetesMonitoringSpec) (*corev1.Secret, error) {
	if kubernetesMonitoringSpec.Enabled &&
		kubernetesMonitoringSpec.CustomProperties != nil &&
		kubernetesMonitoringSpec.CustomProperties.Value != "" &&
		kubernetesMonitoringSpec.CustomProperties.ValueFrom == "" {

		secretName := fmt.Sprintf("%s-%s", name, _const.KubernetesMonitoringCustomPropertiesConfigMapNameSuffix)
		configSecret := &corev1.Secret{}
		err := r.client.Get(context.TODO(), client.ObjectKey{
			Namespace: _const.DynatraceNamespace,
			Name:      secretName,
		}, configSecret)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				configSecret = newConfigSecret(secretName, kubernetesMonitoringSpec.CustomProperties.Value)
				err = r.client.Create(context.TODO(), configSecret)
			}
			return configSecret, err
		}

		secretData := string(configSecret.Data[_const.CustomPropertiesKey])
		if secretData != kubernetesMonitoringSpec.CustomProperties.Value {
			configSecret.Data[_const.CustomPropertiesKey] = []byte(kubernetesMonitoringSpec.CustomProperties.Value)
			err = r.client.Update(context.TODO(), configSecret)
		}
		return configSecret, err
	}
	return nil, nil
}

func newConfigSecret(secretName string, data string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: _const.DynatraceNamespace,
		},
		Data: map[string][]byte{
			_const.CustomPropertiesKey: []byte(data),
		},
	}
}

func (r *ReconcileActiveGate) newStatefulSetForCR(instance *dynatracev1alpha1.DynaKube, kubeSystemUID types.UID) (*appsv1.StatefulSet, error) {

	podSpec, err := builder.BuildActiveGatePodSpecs(instance, kubeSystemUID)
	if err != nil {
		return nil, err
	}
	selectorLabels := builder.BuildLabels(instance.Name, instance.Spec.KubernetesMonitoringSpec.Labels)
	mergedLabels := builder.MergeLabels(instance.Labels, selectorLabels)

	if instance.Spec.KubernetesMonitoringSpec.Enabled {
		mergedLabels = builder.MergeLabels(mergedLabels, instance.Spec.KubernetesMonitoringSpec.Labels)
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        _const.ActivegateName,
			Namespace:   instance.Namespace,
			Labels:      mergedLabels,
			Annotations: map[string]string{},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: instance.Spec.KubernetesMonitoringSpec.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: mergedLabels},
				Spec:       podSpec,
			},
		},
	}

	statefulSetHash, err := generateStatefulSetHash(statefulSet)
	if err != nil {
		return nil, err
	}
	statefulSet.Annotations[annotationTemplateHash] = statefulSetHash

	return statefulSet, nil
}

func generateStatefulSetHash(ds *appsv1.StatefulSet) (string, error) {
	data, err := json.Marshal(ds)
	if err != nil {
		return "", err
	}

	hasher := fnv.New32()
	_, err = hasher.Write(data)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}
