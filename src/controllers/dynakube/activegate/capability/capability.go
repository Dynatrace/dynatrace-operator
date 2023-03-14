package capability

import (
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
)

type baseFunc func() *capabilityBase

var activeGateCapabilities = map[dynatracev1beta1.CapabilityDisplayName]baseFunc{
	dynatracev1beta1.KubeMonCapability.DisplayName:       kubeMonBase,
	dynatracev1beta1.RoutingCapability.DisplayName:       routingBase,
	dynatracev1beta1.MetricsIngestCapability.DisplayName: metricsIngestBase,
	dynatracev1beta1.DynatraceApiCapability.DisplayName:  dynatraceApiBase,
	dynatracev1beta1.SyntheticCapability.DisplayName:     syntheticBase,
}

type Capability interface {
	Enabled() bool
	ShortName() string
	ArgName() string
	Properties() *dynatracev1beta1.CapabilityProperties
}

type capabilityBase struct {
	enabled    bool
	shortName  string
	argName    string
	properties *dynatracev1beta1.CapabilityProperties
}

func (c *capabilityBase) Enabled() bool {
	return c.enabled
}

func (c *capabilityBase) Properties() *dynatracev1beta1.CapabilityProperties {
	return c.properties
}

func (c *capabilityBase) ShortName() string {
	return c.shortName
}

func (c *capabilityBase) ArgName() string {
	return c.argName
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

func syntheticBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.SyntheticCapability.ShortName,
		argName:   dynatracev1beta1.SyntheticCapability.ArgumentName,
	}
	return &c
}

func GenerateActiveGateCapabilities(dynakube *dynatracev1beta1.DynaKube) []Capability {
	return []Capability{
		NewMultiCapability(dynakube),
	}
}

func BuildProxySecretName() string {
	return "dynatrace" + "-" + consts.MultiActiveGateName + "-" + consts.ProxySecretSuffix
}

func BuildServiceName(dynakubeName string, module string) string {
	return dynakubeName + "-" + module
}
