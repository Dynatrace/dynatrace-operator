package routing

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	keyFeature = "feature"
)

func createService(instance *v1alpha1.DynaKube, feature string) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildServiceName(instance.Name, feature),
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{keyFeature: feature},
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       9999,
					TargetPort: intstr.FromInt(9999),
				},
			},
		},
	}
}

func buildServiceName(instanceName string, module string) string {
	return instanceName + "-" + module + "-service"
}
