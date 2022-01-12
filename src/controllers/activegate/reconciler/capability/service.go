package capability

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	statsdingest "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability/statsd-ingest"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/statefulset"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func createService(instance *dynatracev1beta1.DynaKube, feature string) *corev1.Service {
	enableStatsd := instance.NeedsStatsd()
	ports := []corev1.ServicePort{
		{
			Name:       consts.HttpsServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HttpsServicePort,
			TargetPort: intstr.FromString(consts.HttpsServicePortName),
		},
		{
			Name:       consts.HttpServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HttpServicePort,
			TargetPort: intstr.FromString(consts.HttpServicePortName),
		},
	}
	if enableStatsd {
		ports = append(ports, corev1.ServicePort{
			Name:       statsdingest.StatsdIngestPortName,
			Protocol:   corev1.ProtocolUDP,
			Port:       statsdingest.StatsdIngestPort,
			TargetPort: intstr.FromString(statsdingest.StatsdIngestTargetPort),
		})
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildServiceName(instance.Name, feature),
			Namespace: instance.Namespace,
			Labels:    statefulset.BuildLabelsFromInstance(instance, feature),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: statefulset.BuildLabelsFromInstance(instance, feature),
			Ports:    ports,
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
