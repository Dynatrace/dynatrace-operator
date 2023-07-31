package modifiers

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ envModifier = ServicePortModifier{}
var _ builder.Modifier = ServicePortModifier{}

func NewServicePortModifier(dynakube dynatracev1beta1.DynaKube, capability capability.Capability) ServicePortModifier {
	return ServicePortModifier{
		dynakube:   dynakube,
		capability: capability,
	}
}

type ServicePortModifier struct {
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
}

func (mod ServicePortModifier) Enabled() bool {
	return mod.dynakube.NeedsActiveGateServicePorts()
}

func (mod ServicePortModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	baseContainer.ReadinessProbe.HTTPGet.Port = intstr.FromString(consts.HttpsServicePortName)
	baseContainer.Ports = append(baseContainer.Ports, mod.getPorts()...)
	baseContainer.Env = append(baseContainer.Env, mod.getEnvs()...)
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
	return []corev1.EnvVar{
		{
			Name:  consts.EnvDtDnsEntryPoint,
			Value: mod.buildDNSEntryPoint(),
		},
	}
}

func (mod ServicePortModifier) buildDNSEntryPoint() string {
	if mod.capability.ShortName() == consts.MultiActiveGateName && strings.Contains(mod.capability.ArgName(), dynatracev1beta1.RoutingCapability.ArgumentName) ||
		mod.capability.ShortName() == dynatracev1beta1.RoutingCapability.ShortName {
		return fmt.Sprintf("https://%s/communication,https://%s/communication", buildServiceHostName(mod.dynakube.Name, mod.capability.ShortName()), buildServiceDomainName(mod.dynakube.Name, mod.dynakube.Namespace, mod.capability.ShortName()))
	}
	return fmt.Sprintf("https://%s/communication", buildServiceHostName(mod.dynakube.Name, mod.capability.ShortName()))
}

// buildServiceHostName converts the name returned by BuildServiceName
// into the variable name which Kubernetes uses to reference the associated service.
// For more information see: https://kubernetes.io/docs/concepts/services-networking/service/
func buildServiceHostName(dynakubeName string, module string) string {
	serviceName := buildServiceName(dynakubeName, module)
	return fmt.Sprintf("$(%s_SERVICE_HOST):$(%s_SERVICE_PORT)", serviceName, serviceName)
}

func buildServiceDomainName(dynakubeName string, namespaceName string, module string) string {
	return fmt.Sprintf("%s.%s:$(%s_SERVICE_PORT)", capability.BuildServiceName(dynakubeName, module), namespaceName, buildServiceName(dynakubeName, module))
}

func buildServiceName(dynakubeName string, module string) string {
	return strings.ReplaceAll(
		strings.ToUpper(
			capability.BuildServiceName(dynakubeName, module)),
		"-", "_")
}
