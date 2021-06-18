package capability

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/statefulset"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func createService(instance *v1alpha1.DynaKube, feature string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildServiceName(instance.Name, feature),
			Namespace: instance.Namespace,
			Labels:    statefulset.BuildLabelsFromInstance(instance, feature),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: statefulset.BuildLabelsFromInstance(instance, feature),
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       consts.ServicePort,
					TargetPort: intstr.FromString(consts.ServiceTargetPort),
				},
			},
		},
	}
}

func BuildServiceName(instanceName string, module string) string {
	return instanceName + "-" + module
}

// buildServiceHostName converts the name returned by BuildServiceName
// into the variable name which Kubernetes uses to reference the associated service.
// For more information see: https://kubernetes.io/docs/concepts/services-networking/service/
func buildServiceHostName(instanceName string, module string) string {
	serviceName :=
		strings.ReplaceAll(
			strings.ToUpper(
				BuildServiceName(instanceName, module)),
			"-", "_")

	return fmt.Sprintf("$(%s_SERVICE_HOST):$(%s_SERVICE_PORT)", serviceName, serviceName)
}
