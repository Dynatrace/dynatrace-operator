package routing

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type statefulSetProperties struct {
	instance *v1alpha1.DynaKube
}

func newStatefulSetProperties(instance *v1alpha1.DynaKube) *statefulSetProperties {
	return &statefulSetProperties{
		instance: instance,
	}
}

func createStatefulSet(stsProperties *statefulSetProperties) (*appsv1.StatefulSet, error) {
	instance := stsProperties.instance

	sts := &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      instance.Name + StatefulSetSuffix,
			Namespace: instance.Namespace,
		}}
	return sts, nil
}
