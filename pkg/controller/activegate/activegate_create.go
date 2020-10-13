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
)

func (r *ReconcileActiveGate) newStatefulSetForCR(instance *dynatracev1alpha1.ActiveGate, tenantInfo *dtclient.TenantInfo) (*appsv1.StatefulSet, error) {
	podSpec := builder.BuildActiveGatePodSpecs(&instance.Spec, tenantInfo)
	selectorLabels := builder.BuildLabels(instance.GetName(), instance.Spec.Labels)
	mergedLabels := builder.BuildMergeLabels(instance.Labels, selectorLabels)

	sts := &appsv1.StatefulSet{
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

	stsHash, err := generateStatefulSetHash(sts)
	if err != nil {
		return nil, err
	}
	sts.Annotations[annotationTemplateHash] = stsHash

	return sts, nil
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
