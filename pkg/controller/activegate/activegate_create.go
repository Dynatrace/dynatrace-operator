package activegate

import (
	"encoding/json"
	"hash/fnv"
	"strconv"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ReconcileActiveGate) newStatefulSetForCR(instance *dynatracev1alpha1.ActiveGate, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) (*appsv1.StatefulSet, error) {
	podSpec, err := builder.BuildActiveGatePodSpecs(instance, tenantInfo, kubeSystemUID)
	if err != nil {
		return nil, err
	}
	selectorLabels := builder.BuildLabels(instance.GetName(), instance.Spec.Labels)
	mergedLabels := builder.BuildMergeLabels(instance.Labels, selectorLabels)

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Labels:      mergedLabels,
			Annotations: map[string]string{},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: instance.Spec.Replicas,
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
