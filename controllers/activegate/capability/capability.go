package capability

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
)

type Configuration struct {
	SetDnsEntryPoint     bool
	SetReadinessPort     bool
	SetCommunicationPort bool
	CreateService        bool
	ServiceAccountOwner  string
}

type Capability interface {
	GetModuleName() string
	GetCapabilityName() string
	GetProperties() *dynatracev1alpha1.CapabilityProperties
	GetConfiguration() Configuration
}

type capabilityBase struct {
	moduleName     string
	capabilityName string
	properties     *dynatracev1alpha1.CapabilityProperties
	Configuration
}

func (c *capabilityBase) GetProperties() *dynatracev1alpha1.CapabilityProperties {
	return c.properties
}

func (c *capabilityBase) GetConfiguration() Configuration {
	return c.Configuration
}

func (c *capabilityBase) GetModuleName() string {
	return c.moduleName
}

func (c *capabilityBase) GetCapabilityName() string {
	return c.capabilityName
}

func CalculateStatefulSetName(capability Capability, instanceName string) string {
	return instanceName + "-" + capability.GetModuleName()
}

type KubeMonCapability struct {
	capabilityBase
}

type RoutingCapability struct {
	capabilityBase
}

type MetricsCapability struct {
	capabilityBase
}

func NewKubeMonCapability(crProperties *dynatracev1alpha1.CapabilityProperties) *KubeMonCapability {
	return &KubeMonCapability{
		capabilityBase{
			moduleName:     "kubemon",
			capabilityName: "kubernetes_monitoring",
			properties:     crProperties,
			Configuration: Configuration{
				ServiceAccountOwner: "kubernetes-monitoring",
			},
		},
	}
}

func NewRoutingCapability(crProperties *dynatracev1alpha1.CapabilityProperties) *RoutingCapability {
	return &RoutingCapability{
		capabilityBase{
			moduleName:     "routing",
			capabilityName: "MSGrouter",
			properties:     crProperties,
			Configuration: Configuration{
				SetDnsEntryPoint:     true,
				SetReadinessPort:     true,
				SetCommunicationPort: true,
				CreateService:        true,
			},
		},
	}
}

func NewMetricsCapability(crProperties *dynatracev1alpha1.CapabilityProperties) *MetricsCapability {
	return &MetricsCapability{
		capabilityBase{
			moduleName:     "metrics",
			capabilityName: "metrics_ingest",
			properties:     crProperties,
			Configuration: Configuration{
				SetDnsEntryPoint:     true,
				SetReadinessPort:     true,
				SetCommunicationPort: true,
				CreateService:        true,
			},
		},
	}
}
