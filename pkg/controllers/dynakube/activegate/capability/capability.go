package capability

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"k8s.io/utils/net"
)

type baseFunc func() *capabilityBase

var (
	activeGateCapabilities = map[activegate.CapabilityDisplayName]baseFunc{
		activegate.KubeMonCapability.DisplayName:       kubeMonBase,
		activegate.RoutingCapability.DisplayName:       routingBase,
		activegate.MetricsIngestCapability.DisplayName: metricsIngestBase,
		activegate.DynatraceApiCapability.DisplayName:  dynatraceApiBase,
	}
)

type Capability interface {
	Enabled() bool
	ShortName() string
	ArgName() string
	DisplayName() string
	Properties() *activegate.CapabilityProperties
}

type capabilityBase struct {
	properties  *activegate.CapabilityProperties
	shortName   string
	argName     string
	displayName string
	enabled     bool
}

func (capability *capabilityBase) Enabled() bool {
	return capability.enabled
}

func (capability *capabilityBase) Properties() *activegate.CapabilityProperties {
	return capability.properties
}

func (capability *capabilityBase) ShortName() string {
	return capability.shortName
}

func (capability *capabilityBase) ArgName() string {
	return capability.argName
}

func (capability *capabilityBase) DisplayName() string {
	return capability.displayName
}

func CalculateStatefulSetName(capability Capability, dynakubeName string) string {
	return dynakubeName + "-" + capability.ShortName()
}

type MultiCapability struct {
	capabilityBase
}

func NewMultiCapability(dk *dynakube.DynaKube) Capability {
	mc := MultiCapability{
		capabilityBase{
			shortName: consts.MultiActiveGateName,
		},
	}
	if dk == nil || !dk.ActiveGate().IsEnabled() {
		return &mc
	}

	mc.enabled = true
	mc.properties = &dk.Spec.ActiveGate.CapabilityProperties

	if len(dk.Spec.ActiveGate.Capabilities) == 0 && dk.IsExtensionsEnabled() {
		mc.properties.Replicas = 1
	}

	capabilityNames := []string{}
	capabilityDisplayNames := []string{}

	for _, capName := range dk.Spec.ActiveGate.Capabilities {
		capabilityGenerator, ok := activeGateCapabilities[capName]
		if !ok {
			continue
		}

		capGen := capabilityGenerator()
		capabilityNames = append(capabilityNames, capGen.argName)
		capabilityDisplayNames = append(capabilityDisplayNames, capGen.displayName)
	}

	if dk.IsExtensionsEnabled() {
		capabilityNames = append(capabilityNames, "extension_controller")
		capabilityDisplayNames = append(capabilityDisplayNames, "extension_controller")
	}

	mc.argName = strings.Join(capabilityNames, ",")
	mc.displayName = strings.Join(capabilityDisplayNames, ", ")

	return &mc
}

func kubeMonBase() *capabilityBase {
	c := capabilityBase{
		shortName:   activegate.KubeMonCapability.ShortName,
		argName:     activegate.KubeMonCapability.ArgumentName,
		displayName: string(activegate.KubeMonCapability.DisplayName),
	}

	return &c
}

func routingBase() *capabilityBase {
	c := capabilityBase{
		shortName:   activegate.RoutingCapability.ShortName,
		argName:     activegate.RoutingCapability.ArgumentName,
		displayName: string(activegate.RoutingCapability.DisplayName),
	}

	return &c
}

func metricsIngestBase() *capabilityBase {
	c := capabilityBase{
		shortName:   activegate.MetricsIngestCapability.ShortName,
		argName:     activegate.MetricsIngestCapability.ArgumentName,
		displayName: string(activegate.MetricsIngestCapability.DisplayName),
	}

	return &c
}

func dynatraceApiBase() *capabilityBase {
	c := capabilityBase{
		shortName:   activegate.DynatraceApiCapability.ShortName,
		argName:     activegate.DynatraceApiCapability.ArgumentName,
		displayName: string(activegate.DynatraceApiCapability.DisplayName),
	}

	return &c
}

func GenerateActiveGateCapabilities(dk *dynakube.DynaKube) []Capability {
	return []Capability{
		NewMultiCapability(dk),
	}
}

func BuildServiceName(dynakubeName string, module string) string {
	return dynakubeName + "-" + module
}

func BuildDNSEntryPointWithoutEnvVars(dynakubeName, dynakubeNamespace string, capability Capability) string {
	return fmt.Sprintf("%s.%s", BuildServiceName(dynakubeName, capability.ShortName()), dynakubeNamespace)
}

func BuildDNSEntryPoint(dk dynakube.DynaKube, capability Capability) string {
	entries := []string{}

	for _, ip := range dk.Status.ActiveGate.ServiceIPs {
		if net.IsIPv6String(ip) {
			ip = "[" + ip + "]"
		}

		serviceHostEntry := buildDNSEntry(buildServiceHostName(ip))
		entries = append(entries, serviceHostEntry)
	}

	if dk.ActiveGate().IsRoutingEnabled() {
		serviceDomain := buildServiceDomainName(dk.Name, dk.Namespace, capability.ShortName())
		serviceDomainEntry := buildDNSEntry(serviceDomain)
		entries = append(entries, serviceDomainEntry)
	}

	return strings.Join(entries, ",")
}

func buildServiceHostName(host string) string {
	return fmt.Sprintf("%s:%d", host, consts.HttpsServicePort)
}

func buildServiceDomainName(dynakubeName string, namespaceName string, module string) string {
	return fmt.Sprintf("%s.%s:%d", BuildServiceName(dynakubeName, module), namespaceName, consts.HttpsServicePort)
}

func buildDNSEntry(host string) string {
	return fmt.Sprintf("https://%s/communication", host)
}
