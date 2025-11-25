package capability

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateService(dk *dynakube.DynaKube) *corev1.Service {
	var ports []corev1.ServicePort

	ports = append(ports,
		corev1.ServicePort{
			Name:       consts.HTTPSServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HTTPSServicePort,
			TargetPort: intstr.FromString(consts.HTTPSServicePortName),
		}, corev1.ServicePort{
			Name:       consts.HTTPServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HTTPServicePort,
			TargetPort: intstr.FromString(consts.HTTPServicePortName),
		})

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.ActiveGateComponentLabel)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capability.BuildServiceName(dk.Name),
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
	appLabels := k8slabel.NewAppLabels(k8slabel.ActiveGateComponentLabel, dynakubeName, "", "")

	return appLabels.BuildMatchLabels()
}
