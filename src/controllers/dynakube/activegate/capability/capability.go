package capability

import (
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
)

const (
	SyntheticName                      = "synthetic"
	SyntheticActiveGateEnvCapabilities = "synthetic,beacon_forwarder,beacon_forwarder_synthetic"
)

type baseFunc func() *capabilityBase

var activeGateCapabilities = map[dynatracev1beta1.CapabilityDisplayName]baseFunc{
	dynatracev1beta1.KubeMonCapability.DisplayName:       kubeMonBase,
	dynatracev1beta1.RoutingCapability.DisplayName:       routingBase,
	dynatracev1beta1.MetricsIngestCapability.DisplayName: metricsIngestBase,
	dynatracev1beta1.DynatraceApiCapability.DisplayName:  dynatraceApiBase,
}

type Capability interface {
	Enabled() bool
	ShortName() string
	ArgName() string
	Properties() *dynatracev1beta1.CapabilityProperties
	IsSynthetic() bool
}

type capabilityBase struct {
	enabled    bool
	shortName  string
	argName    string
	properties *dynatracev1beta1.CapabilityProperties
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

func (capability *capabilityBase) IsSynthetic() bool {
	return capability.shortName == SyntheticName
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

type SyntheticCapability struct {
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

func NewSyntheticCapability(dynaKube *dynatracev1beta1.DynaKube) *SyntheticCapability {
	capability := &SyntheticCapability{
		*syntheticBase(),
	}
	if dynaKube == nil {
		return capability
	}
	capability.enabled = dynaKube.IsSyntheticMonitoringEnabled()
	capability.properties = &dynatracev1beta1.CapabilityProperties{
		Image: dynaKube.Spec.ActiveGate.CapabilityProperties.Image,
		Env:   dynaKube.Spec.ActiveGate.CapabilityProperties.Env,
	}
	return capability
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
		shortName: SyntheticName,
		argName:   SyntheticActiveGateEnvCapabilities,
	}
	return &c
}

func GenerateActiveGateCapabilities(dynaKube *dynatracev1beta1.DynaKube) []Capability {
	return []Capability{
		NewKubeMonCapability(dynaKube),
		NewRoutingCapability(dynaKube),
		NewMultiCapability(dynaKube),
		NewSyntheticCapability(dynaKube),
	}
}

func BuildProxySecretName() string {
	return "dynatrace" + "-" + consts.MultiActiveGateName + "-" + consts.ProxySecretSuffix
}

func BuildServiceName(dynakubeName string, module string) string {
	return dynakubeName + "-" + module
}
