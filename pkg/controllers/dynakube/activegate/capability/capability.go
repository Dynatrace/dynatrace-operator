package capability

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"k8s.io/utils/net"
)

type baseFunc func() *capabilityBase

var (
	activeGateCapabilities = map[dynakube.CapabilityDisplayName]baseFunc{
		dynakube.KubeMonCapability.DisplayName:       kubeMonBase,
		dynakube.RoutingCapability.DisplayName:       routingBase,
		dynakube.MetricsIngestCapability.DisplayName: metricsIngestBase,
		dynakube.DynatraceApiCapability.DisplayName:  dynatraceApiBase,
	}
)

type Capability interface {
	Enabled() bool
	ShortName() string
	ArgName() string
	DisplayName() string
	Properties() *dynakube.CapabilityProperties
}

type capabilityBase struct {
	properties  *dynakube.CapabilityProperties
	shortName   string
	argName     string
	displayName string
	enabled     bool
}

func (capability *capabilityBase) Enabled() bool {
	return capability.enabled
}

func (capability *capabilityBase) Properties() *dynakube.CapabilityProperties {
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
	if dk == nil || !dk.ActiveGateMode() {
		return &mc
	}

	mc.enabled = true
	mc.properties = &dk.Spec.ActiveGate.CapabilityProperties
	capabilityNames := []string{}
	capabilityDisplayNames := make([]string, len(dk.Spec.ActiveGate.Capabilities))

	for i, capName := range dk.Spec.ActiveGate.Capabilities {
		capabilityGenerator, ok := activeGateCapabilities[capName]
		if !ok {
			continue
		}

		capGen := capabilityGenerator()
		capabilityNames = append(capabilityNames, capGen.argName)
		capabilityDisplayNames[i] = capGen.displayName
	}

	mc.argName = strings.Join(capabilityNames, ",")
	mc.displayName = strings.Join(capabilityDisplayNames, ", ")

	return &mc
}

func kubeMonBase() *capabilityBase {
	c := capabilityBase{
		shortName:   dynakube.KubeMonCapability.ShortName,
		argName:     dynakube.KubeMonCapability.ArgumentName,
		displayName: string(dynakube.KubeMonCapability.DisplayName),
	}

	return &c
}

func routingBase() *capabilityBase {
	c := capabilityBase{
		shortName:   dynakube.RoutingCapability.ShortName,
		argName:     dynakube.RoutingCapability.ArgumentName,
		displayName: string(dynakube.RoutingCapability.DisplayName),
	}

	return &c
}

func metricsIngestBase() *capabilityBase {
	c := capabilityBase{
		shortName:   dynakube.MetricsIngestCapability.ShortName,
		argName:     dynakube.MetricsIngestCapability.ArgumentName,
		displayName: string(dynakube.MetricsIngestCapability.DisplayName),
	}

	return &c
}

func dynatraceApiBase() *capabilityBase {
	c := capabilityBase{
		shortName:   dynakube.DynatraceApiCapability.ShortName,
		argName:     dynakube.DynatraceApiCapability.ArgumentName,
		displayName: string(dynakube.DynatraceApiCapability.DisplayName),
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

	if dk.IsRoutingActiveGateEnabled() {
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
