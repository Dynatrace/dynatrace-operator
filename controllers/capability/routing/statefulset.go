package routing

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	annotationTemplateHash    = "internal.operator.dynatrace.com/template-hash"
	annotationImageHash       = "internal.operator.dynatrace.com/image-hash"
	annotationImageVersion    = "internal.operator.dynatrace.com/image-version"
	annotationCustomPropsHash = "internal.operator.dynatrace.com/custom-properties-hash"
)

type statefulSetProperties struct {
	*v1alpha1.DynaKube
	*v1alpha1.CapabilityProperties
	CustomPropertiesHash string
	KubeSystemUID        types.UID
}

func newStatefulSetProperties(instance *v1alpha1.DynaKube, capabilityProperties *v1alpha1.CapabilityProperties, kubeSystemUID types.UID, customPropertiesHash string) *statefulSetProperties {
	return &statefulSetProperties{
		DynaKube:             instance,
		CapabilityProperties: capabilityProperties,
		CustomPropertiesHash: customPropertiesHash,
		KubeSystemUID:        kubeSystemUID,
	}
}

func createStatefulSet(stsProperties *statefulSetProperties) (*appsv1.StatefulSet, error) {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsProperties.Name + StatefulSetSuffix,
			Namespace: stsProperties.Namespace,
			Labels:    capability.BuildLabels(stsProperties.DynaKube, stsProperties.CapabilityProperties),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            stsProperties.Replicas,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector:            &metav1.LabelSelector{MatchLabels: capability.BuildLabelsFromInstance(stsProperties.DynaKube)},
		}}
	return sts, nil
}
