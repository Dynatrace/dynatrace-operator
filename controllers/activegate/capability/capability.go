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

type Capability struct {
	ModuleName     string
	CapabilityName string
	Properties     *dynatracev1alpha1.CapabilityProperties
	Configuration
}

func (c *Capability) CalculateStatefulSetName(instanceName string) string {
	return instanceName + "-" + c.ModuleName
}

type CapabilityType int

const (
	Kubemon CapabilityType = iota
	Routing
	Metrics
)

func MakeCapapability(ct CapabilityType, crProperties *dynatracev1alpha1.CapabilityProperties) *Capability {
	if crProperties == nil {
		return nil
	}

	switch ct {
	case Kubemon:
		return &Capability{
			ModuleName:     "kubemon",
			CapabilityName: "kubernetes_monitoring",
			Properties:     crProperties,
			Configuration: Configuration{
				ServiceAccountOwner: "kubernetes-monitoring",
			},
		}
	case Routing:
		return &Capability{
			ModuleName:     "routing",
			CapabilityName: "MSGrouter",
			Properties:     crProperties,
			Configuration: Configuration{
				SetDnsEntryPoint:     true,
				SetReadinessPort:     true,
				SetCommunicationPort: true,
				CreateService:        true,
			},
		}
	case Metrics:
		return &Capability{
			ModuleName:     "metrics",
			CapabilityName: "metrics_ingest",
			Properties:     crProperties,
			Configuration: Configuration{
				SetDnsEntryPoint:     true,
				SetReadinessPort:     true,
				SetCommunicationPort: true,
				CreateService:        true,
			},
		}
	}

	return nil
}
