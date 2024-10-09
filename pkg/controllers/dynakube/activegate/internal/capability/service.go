package capability

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateService(dk *dynakube.DynaKube, feature string) *corev1.Service {
	var ports []corev1.ServicePort

	if dk.ActiveGate().NeedsService() {
		ports = append(ports,
			corev1.ServicePort{
				Name:       consts.HttpsServicePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       consts.HttpsServicePort,
				TargetPort: intstr.FromString(consts.HttpsServicePortName),
			},
		)
		if dk.ActiveGate().IsMetricsIngestEnabled() {
			ports = append(ports, corev1.ServicePort{
				Name:       consts.HttpServicePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       consts.HttpServicePort,
				TargetPort: intstr.FromString(consts.HttpServicePortName),
			})
		}
	}

	coreLabels := labels.NewCoreLabels(dk.Name, labels.ActiveGateComponentLabel)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.BuildServiceName(dk.Name, feature),
			Namespace: dk.Namespace,
			Labels:    coreLabels.BuildLabels(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: buildSelectorLabels(dk.Name),
			Ports:    ports,
		},
	}
}

func buildSelectorLabels(dynakubeName string) map[string]string {
	appLabels := labels.NewAppLabels(labels.ActiveGateComponentLabel, dynakubeName, "", "")

	return appLabels.BuildMatchLabels()
}
