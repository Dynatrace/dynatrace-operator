package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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

func NewServicePortModifier(dk dynakube.DynaKube, capability capability.Capability, envMap *prioritymap.Map) ServicePortModifier {
	return ServicePortModifier{
		dk:         dk,
		capability: capability,
		envMap:     envMap,
	}
}

type ServicePortModifier struct {
	capability capability.Capability
	envMap     *prioritymap.Map
	dk         dynakube.DynaKube
}

func (mod ServicePortModifier) Enabled() bool {
	return mod.dk.ActiveGate().NeedsService()
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
	if mod.dk.ActiveGate().IsMetricsIngestEnabled() {
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
	return capability.BuildDNSEntryPoint(mod.dk, mod.capability)
}
