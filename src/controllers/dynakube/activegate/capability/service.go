package capability

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func createService(dynakube *dynatracev1beta1.DynaKube, feature string, servicePorts AgServicePorts) *corev1.Service {
	var ports []corev1.ServicePort

	if servicePorts.Webserver {
		ports = append(ports,
			corev1.ServicePort{
				Name:       HttpsServicePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       HttpsServicePort,
				TargetPort: intstr.FromString(HttpsServicePortName),
			},
			corev1.ServicePort{
				Name:       HttpServicePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       HttpServicePort,
				TargetPort: intstr.FromString(HttpServicePortName),
			},
		)
	}

	if servicePorts.Statsd {
		ports = append(ports,
			corev1.ServicePort{
				Name:       consts.StatsdIngestPortName,
				Protocol:   corev1.ProtocolUDP,
				Port:       consts.StatsdIngestPort,
				TargetPort: intstr.FromString(consts.StatsdIngestTargetPort),
			},
		)
	}

	coreLabels := kubeobjects.NewCoreLabels(dynakube.Name, kubeobjects.ActiveGateComponentLabel)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildServiceName(dynakube.Name, feature),
			Namespace: dynakube.Namespace,
			Labels:    coreLabels.BuildLabels(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: buildSelectorLabels(dynakube.Name),
			Ports:    ports,
		},
	}
}

func BuildServiceName(dynakubeName string, module string) string {
	return dynakubeName + "-" + module
}

// buildServiceHostName converts the name returned by BuildServiceName
// into the variable name which Kubernetes uses to reference the associated service.
// For more information see: https://kubernetes.io/docs/concepts/services-networking/service/
func buildServiceHostName(dynakubeName string, module string) string {
	serviceName :=
		strings.ReplaceAll(
			strings.ToUpper(
				BuildServiceName(dynakubeName, module)),
			"-", "_")

	return fmt.Sprintf("$(%s_SERVICE_HOST):$(%s_SERVICE_PORT)", serviceName, serviceName)
}

func buildSelectorLabels(dynakubeName string) map[string]string {
	appLabels := kubeobjects.NewAppLabels(kubeobjects.ActiveGateComponentLabel, dynakubeName, "", "")
	return appLabels.BuildMatchLabels()
}
