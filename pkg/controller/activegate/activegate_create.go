package activegate

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"strconv"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/builder"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileActiveGate) createCustomPropertiesConfigMap(kubernetesMonitoringSpec *dynatracev1alpha1.KubernetesMonitoringSpec) error {
	if kubernetesMonitoringSpec.Enabled &&
		kubernetesMonitoringSpec.CustomProperties != nil &&
		kubernetesMonitoringSpec.CustomProperties.Value != "" &&
		kubernetesMonitoringSpec.CustomProperties.ValueFrom == "" {

		configMap := &corev1.ConfigMap{}
		err := r.client.Get(context.TODO(), client.ObjectKey{
			Namespace: _const.DynatraceNamespace,
			Name:      _const.CustomPropertiesConfigMapName,
		}, configMap)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				configMap = &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      _const.CustomPropertiesConfigMapName,
						Namespace: _const.DynatraceNamespace,
					},
					Data: map[string]string{
						_const.CustomPropertiesKey: kubernetesMonitoringSpec.CustomProperties.Value,
					},
				}
				err = r.client.Create(context.TODO(), configMap)
			}
			return err
		}
	}
	return nil
}

func (r *ReconcileActiveGate) newStatefulSetForCR(instance *dynatracev1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) (*appsv1.StatefulSet, error) {

	podSpec, err := builder.BuildActiveGatePodSpecs(instance, tenantInfo, kubeSystemUID)
	if err != nil {
		return nil, err
	}
	selectorLabels := builder.BuildLabels(instance.Name, instance.Spec.KubernetesMonitoringSpec.Labels)
	mergedLabels := builder.MergeLabels(instance.Labels, selectorLabels)

	if instance.Spec.KubernetesMonitoringSpec.Enabled {
		mergedLabels = builder.BuildMergeLabels(mergedLabels,
			instance.Spec.KubernetesMonitoringSpec.Labels)
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
