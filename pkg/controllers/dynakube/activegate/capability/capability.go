package capability

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
)

type baseFunc func() *capabilityBase

var (
	activeGateCapabilities = map[dynatracev1beta1.CapabilityDisplayName]baseFunc{
		dynatracev1beta1.KubeMonCapability.DisplayName:       kubeMonBase,
		dynatracev1beta1.RoutingCapability.DisplayName:       routingBase,
		dynatracev1beta1.MetricsIngestCapability.DisplayName: metricsIngestBase,
		dynatracev1beta1.DynatraceApiCapability.DisplayName:  dynatraceApiBase,
	}
)

type Capability interface {
	Enabled() bool
	ShortName() string
	ArgName() string
	Properties() *dynatracev1beta1.CapabilityProperties
}

type capabilityBase struct {
	properties *dynatracev1beta1.CapabilityProperties
	shortName  string
	argName    string
	enabled    bool
}

func (capability *capabilityBase) Enabled() bool {
	return capability.enabled
}

func (capability *capabilityBase) Properties() *dynatracev1beta1.CapabilityProperties {
	return capability.properties
}

func (capability *capabilityBase) ShortName() string {
	return capability.shortName
}

func (capability *capabilityBase) ArgName() string {
	return capability.argName
}

func CalculateStatefulSetName(capability Capability, dynakubeName string) string {
	return dynakubeName + "-" + capability.ShortName()
}

// Deprecated: Use MultiCapability instead
type KubeMonCapability struct {
	capabilityBase
}

// Deprecated: Use MultiCapability instead
type RoutingCapability struct {
	capabilityBase
}

type MultiCapability struct {
	capabilityBase
}

func NewMultiCapability(dk *dynatracev1beta1.DynaKube) *MultiCapability {
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

	for _, capName := range dk.Spec.ActiveGate.Capabilities {
		capabilityGenerator, ok := activeGateCapabilities[capName]
		if !ok {
			continue
		}

		capGen := capabilityGenerator()
		capabilityNames = append(capabilityNames, capGen.argName)
	}

	mc.argName = strings.Join(capabilityNames, ",")

	return &mc
}

// Deprecated
func NewKubeMonCapability(dk *dynatracev1beta1.DynaKube) *KubeMonCapability {
	c := &KubeMonCapability{
		*kubeMonBase(),
	}
	if dk == nil {
		return c
	}

	c.enabled = dk.Spec.KubernetesMonitoring.Enabled
	c.properties = &dk.Spec.KubernetesMonitoring.CapabilityProperties

	return c
}

// Deprecated
func NewRoutingCapability(dk *dynatracev1beta1.DynaKube) *RoutingCapability {
	c := &RoutingCapability{
		*routingBase(),
	}
	if dk == nil {
		return c
	}

	c.enabled = dk.Spec.Routing.Enabled
	c.properties = &dk.Spec.Routing.CapabilityProperties

	return c
}

func kubeMonBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.KubeMonCapability.ShortName,
		argName:   dynatracev1beta1.KubeMonCapability.ArgumentName,
	}

	return &c
}

func routingBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.RoutingCapability.ShortName,
		argName:   dynatracev1beta1.RoutingCapability.ArgumentName,
	}

	return &c
}

func metricsIngestBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.MetricsIngestCapability.ShortName,
		argName:   dynatracev1beta1.MetricsIngestCapability.ArgumentName,
	}

	return &c
}

func dynatraceApiBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.DynatraceApiCapability.ShortName,
		argName:   dynatracev1beta1.DynatraceApiCapability.ArgumentName,
	}

	return &c
}

func GenerateActiveGateCapabilities(dk *dynatracev1beta1.DynaKube) []Capability {
	return []Capability{
		NewKubeMonCapability(dk),
		NewRoutingCapability(dk),
		NewMultiCapability(dk),
	}
}

func BuildServiceName(dynakubeName string, module string) string {
	return dynakubeName + "-" + module
}

func BuildDNSEntryPointWithoutEnvVars(dynakubeName, dynakubeNamespace string, capability Capability) string {
	return fmt.Sprintf("%s.%s", BuildServiceName(dynakubeName, capability.ShortName()), dynakubeNamespace)
}

func BuildDNSEntryPoint(dk dynatracev1beta1.DynaKube, capability Capability) string {
	entries := []string{}

	serviceHost := buildServiceHostName(dk.Name, capability.ShortName())
	if dk.Spec.ActiveGate.IPv6 {
		serviceHost = "[" + serviceHost + "]"
	}

	serviceHostEntry := buildDNSEntry(serviceHost)
	entries = append(entries, serviceHostEntry)

	if dk.IsRoutingActiveGateEnabled() {
		serviceDomain := buildServiceDomainName(dk.Name, dk.Namespace, capability.ShortName())
		serviceDomainEntry := buildDNSEntry(serviceDomain)
		entries = append(entries, serviceDomainEntry)
	}

	return strings.Join(entries, ",")
}

// buildServiceHostName converts the name returned by BuildServiceName
// into the variable name which Kubernetes uses to reference the associated service.
// For more information see: https://kubernetes.io/docs/concepts/services-networking/service/
func buildServiceHostName(dynakubeName string, module string) string {
	serviceName := buildServiceNameUnderscore(dynakubeName, module)

	return fmt.Sprintf("$(%s_SERVICE_HOST):$(%s_SERVICE_PORT)", serviceName, serviceName)
}

// buildServiceDomainName builds service domain name
func buildServiceDomainName(dynakubeName string, namespaceName string, module string) string {
	return fmt.Sprintf("%s.%s:$(%s_SERVICE_PORT)", BuildServiceName(dynakubeName, module), namespaceName, buildServiceNameUnderscore(dynakubeName, module))
}

// buildServiceNameUnderscore converts result of BuildServiceName by replacing dashes with underscores
// to make it env variable compatible because it's only special symbol it supports
func buildServiceNameUnderscore(dynakubeName string, module string) string {
	return strings.ReplaceAll(
		strings.ToUpper(
			BuildServiceName(dynakubeName, module)),
		"-", "_")
}

func buildDNSEntry(host string) string {
	return fmt.Sprintf("https://%s/communication", host)
}
