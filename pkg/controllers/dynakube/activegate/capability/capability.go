package capability

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"k8s.io/utils/net"
	"k8s.io/utils/ptr"
)

var (
	activeGateCapabilities = map[activegate.CapabilityDisplayName]string{
		activegate.KubeMonCapability.DisplayName:       activegate.KubeMonCapability.ArgumentName,
		activegate.RoutingCapability.DisplayName:       activegate.RoutingCapability.ArgumentName,
		activegate.MetricsIngestCapability.DisplayName: activegate.MetricsIngestCapability.ArgumentName,
		activegate.DynatraceApiCapability.DisplayName:  activegate.DynatraceApiCapability.ArgumentName,
		activegate.DebuggingCapability.DisplayName:     activegate.DebuggingCapability.ArgumentName,
	}
)

type Capability interface {
	Enabled() bool
	ArgName() string
	Properties() *activegate.CapabilityProperties
}

type capabilityBase struct {
	properties *activegate.CapabilityProperties
	argName    string
}

func (capability *capabilityBase) Enabled() bool {
	return len(capability.argName) > 0
}

func (capability *capabilityBase) Properties() *activegate.CapabilityProperties {
	return capability.properties
}

func (capability *capabilityBase) ArgName() string {
	return capability.argName
}

func CalculateStatefulSetName(dynakubeName string) string {
	return dynakubeName + "-" + consts.MultiActiveGateName
}

type MultiCapability struct {
	capabilityBase
}

func NewMultiCapability(dk *dynakube.DynaKube) Capability {
	mc := MultiCapability{
		capabilityBase{},
	}
	if dk == nil || !dk.ActiveGate().IsEnabled() {
		return &mc
	}

	mc.properties = &dk.Spec.ActiveGate.CapabilityProperties

	if len(dk.Spec.ActiveGate.Capabilities) == 0 && dk.IsExtensionsEnabled() {
		mc.properties.Replicas = ptr.To(int32(1))
	}

	capabilityNames := []string{}

	for _, capName := range dk.Spec.ActiveGate.Capabilities {
		argName, ok := activeGateCapabilities[capName]
		if !ok {
			continue
		}

		capabilityNames = append(capabilityNames, argName)
	}

	if dk.IsExtensionsEnabled() {
		capabilityNames = append(capabilityNames, "extension_controller")
	}

	if dk.TelemetryIngest().IsEnabled() {
		capabilityNames = append(capabilityNames, "log_analytics_collector", "generic_ingest", "otlp_ingest")
	}

	mc.argName = strings.Join(capabilityNames, ",")

	return &mc
}

func GenerateActiveGateCapabilities(dk *dynakube.DynaKube) []Capability {
	return []Capability{
		NewMultiCapability(dk),
	}
}

func BuildServiceName(dynakubeName string) string {
	return dynakubeName + "-" + consts.MultiActiveGateName
}

// BuilDNSEntryPoint will create a string listing of the full DNS entry points for the Service of the ActiveGate in the provided DynaKube
// example: https://34.118.233.238:443,https://dynakube-activegate.dynatrace:443
func BuildDNSEntryPoint(dk dynakube.DynaKube) string {
	entries := []string{}

	for _, ip := range dk.Status.ActiveGate.ServiceIPs {
		if net.IsIPv6String(ip) {
			ip = "[" + ip + "]"
		}

		serviceHostEntry := buildDNSEntry(buildServiceHostName(ip))
		entries = append(entries, serviceHostEntry)
	}

	if dk.ActiveGate().IsRoutingEnabled() {
		serviceDomain := buildServiceDomainName(dk.Name, dk.Namespace)
		serviceDomainEntry := buildDNSEntry(serviceDomain)
		entries = append(entries, serviceDomainEntry)
	}

	return strings.Join(entries, ",")
}

// BuildHostEntries will create a string listing the host entries for the Service of the ActiveGate in the provided DynaKube
// Meant to be used as a NO_PROXY value for components needing to directly communicate with the ActiveGate.
// example: 34.118.233.238,dynakube-activegate.dynatrace
func BuildHostEntries(dk dynakube.DynaKube) string {
	entries := []string{}

	for _, ip := range dk.Status.ActiveGate.ServiceIPs {
		if net.IsIPv6String(ip) {
			ip = "[" + ip + "]"
		}

		entries = append(entries, ip)
	}

	if dk.ActiveGate().IsRoutingEnabled() {
		entries = append(entries, fmt.Sprintf("%s.%s", BuildServiceName(dk.Name), dk.Namespace))
	}

	return strings.Join(entries, ",")
}

func buildServiceHostName(host string) string {
	return fmt.Sprintf("%s:%d", host, consts.HttpsServicePort)
}

func buildServiceDomainName(dynakubeName string, namespaceName string) string {
	return fmt.Sprintf("%s.%s:%d", BuildServiceName(dynakubeName), namespaceName, consts.HttpsServicePort)
}

func buildDNSEntry(host string) string {
	return fmt.Sprintf("https://%s/communication", host)
}
