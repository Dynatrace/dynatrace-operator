package capability

import (
	"fmt"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"k8s.io/utils/net"
)

type baseFunc func() *capabilityBase

var (
	activeGateCapabilities = map[dynatracev1beta2.CapabilityDisplayName]baseFunc{
		dynatracev1beta2.KubeMonCapability.DisplayName:       kubeMonBase,
		dynatracev1beta2.RoutingCapability.DisplayName:       routingBase,
		dynatracev1beta2.MetricsIngestCapability.DisplayName: metricsIngestBase,
		dynatracev1beta2.DynatraceApiCapability.DisplayName:  dynatraceApiBase,
	}
)

type Capability interface {
	Enabled() bool
	ShortName() string
	ArgName() string
	DisplayName() string
	Properties() *dynatracev1beta2.CapabilityProperties
}

type capabilityBase struct {
	properties  *dynatracev1beta2.CapabilityProperties
	shortName   string
	argName     string
	displayName string
	enabled     bool
}

func (capability *capabilityBase) Enabled() bool {
	return capability.enabled
}

func (capability *capabilityBase) Properties() *dynatracev1beta2.CapabilityProperties {
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

func NewMultiCapability(dk *dynatracev1beta2.DynaKube) Capability {
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
		shortName:   dynatracev1beta2.KubeMonCapability.ShortName,
		argName:     dynatracev1beta2.KubeMonCapability.ArgumentName,
		displayName: string(dynatracev1beta2.KubeMonCapability.DisplayName),
	}

	return &c
}

func routingBase() *capabilityBase {
	c := capabilityBase{
		shortName:   dynatracev1beta2.RoutingCapability.ShortName,
		argName:     dynatracev1beta2.RoutingCapability.ArgumentName,
		displayName: string(dynatracev1beta2.RoutingCapability.DisplayName),
	}

	return &c
}

func metricsIngestBase() *capabilityBase {
	c := capabilityBase{
		shortName:   dynatracev1beta2.MetricsIngestCapability.ShortName,
		argName:     dynatracev1beta2.MetricsIngestCapability.ArgumentName,
		displayName: string(dynatracev1beta2.MetricsIngestCapability.DisplayName),
	}

	return &c
}

func dynatraceApiBase() *capabilityBase {
	c := capabilityBase{
		shortName:   dynatracev1beta2.DynatraceApiCapability.ShortName,
		argName:     dynatracev1beta2.DynatraceApiCapability.ArgumentName,
		displayName: string(dynatracev1beta2.DynatraceApiCapability.DisplayName),
	}

	return &c
}

func GenerateActiveGateCapabilities(dk *dynatracev1beta2.DynaKube) []Capability {
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

func BuildDNSEntryPoint(dk dynatracev1beta2.DynaKube, capability Capability) string {
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
