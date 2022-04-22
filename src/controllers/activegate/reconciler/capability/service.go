package capability

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func createService(instance *dynatracev1beta1.DynaKube, feature string, servicePorts capability.AgServicePorts) *corev1.Service {
	var ports []corev1.ServicePort

	if servicePorts.Webserver {
		ports = append(ports,
			corev1.ServicePort{
				Name:       capability.HttpsServicePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       capability.HttpsServicePort,
				TargetPort: intstr.FromString(capability.HttpsServicePortName),
			},
			corev1.ServicePort{
				Name:       capability.HttpServicePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       capability.HttpServicePort,
				TargetPort: intstr.FromString(capability.HttpServicePortName),
			},
		)
	}

	if servicePorts.Statsd {
		ports = append(ports,
			corev1.ServicePort{
				Name:       capability.StatsdIngestPortName,
				Protocol:   corev1.ProtocolUDP,
				Port:       capability.StatsdIngestPort,
				TargetPort: intstr.FromString(capability.StatsdIngestTargetPort),
			},
		)
	}

	coreLabels := kubeobjects.NewCoreLabels(instance.Name, kubeobjects.ActiveGateComponentLabel)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildServiceName(instance.Name, feature),
			Namespace: instance.Namespace,
			Labels:    coreLabels.BuildLabels(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: coreLabels.BuildMatchLabels(),
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
