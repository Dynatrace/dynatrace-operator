package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ envModifier = ServicePortModifier{}
var _ builder.Modifier = ServicePortModifier{}

func NewServicePortModifier(dynakube dynatracev1beta1.DynaKube, capability capability.Capability, envMap *prioritymap.Map) ServicePortModifier {
	return ServicePortModifier{
		dynakube:   dynakube,
		capability: capability,
		envMap:     envMap,
	}
}

type ServicePortModifier struct {
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
	envMap     *prioritymap.Map
}

func (mod ServicePortModifier) Enabled() bool {
	return mod.dynakube.NeedsActiveGateServicePorts()
}

func (mod ServicePortModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := container.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.ReadinessProbe.HTTPGet.Port = intstr.FromString(consts.HttpsServicePortName)
	baseContainer.Ports = append(baseContainer.Ports, mod.getPorts()...)
	baseContainer.Env = mod.getEnvs()
	return nil
}

func (mod ServicePortModifier) getPorts() []corev1.ContainerPort {
	ports := []corev1.ContainerPort{
		{
			Name:          consts.HttpsServicePortName,
			ContainerPort: consts.HttpsContainerPort,
		},
	}
	if mod.dynakube.IsMetricsIngestActiveGateEnabled() {
		ports = append(ports, corev1.ContainerPort{
			Name:          consts.HttpServicePortName,
			ContainerPort: consts.HttpContainerPort,
		})
	}
	return ports
}

func (mod ServicePortModifier) getEnvs() []corev1.EnvVar {
	prioritymap.Append(mod.envMap,
		[]corev1.EnvVar{
			{
				Name:  consts.EnvDtDnsEntryPoint,
				Value: mod.buildDNSEntryPoint(),
			},
		},
		prioritymap.WithPriority(modifierEnvPriority))
	return mod.envMap.AsEnvVars()
}

func (mod ServicePortModifier) buildDNSEntryPoint() string {
	return capability.BuildDNSEntryPoint(mod.dynakube.Name, mod.dynakube.Namespace, mod.capability)
}
