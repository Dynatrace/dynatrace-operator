package capability

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateService(dynakube *dynatracev1beta1.DynaKube, feature string) *corev1.Service {
	var ports []corev1.ServicePort

	if dynakube.NeedsActiveGateServicePorts() {
		ports = append(ports,
			corev1.ServicePort{
				Name:       consts.HttpsServicePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       consts.HttpsServicePort,
				TargetPort: intstr.FromString(consts.HttpsServicePortName),
			},
		)
		if dynakube.IsMetricsIngestActiveGateEnabled() {
			ports = append(ports, corev1.ServicePort{
				Name:       consts.HttpServicePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       consts.HttpServicePort,
				TargetPort: intstr.FromString(consts.HttpServicePortName),
			})
		}
	}

	coreLabels := kubeobjects.NewCoreLabels(dynakube.Name, kubeobjects.ActiveGateComponentLabel)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.BuildServiceName(dynakube.Name, feature),
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

func buildSelectorLabels(dynakubeName string) map[string]string {
	appLabels := kubeobjects.NewAppLabels(kubeobjects.ActiveGateComponentLabel, dynakubeName, "", "")
	return appLabels.BuildMatchLabels()
}
