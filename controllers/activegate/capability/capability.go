package capability

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

const (
	trustStoreVolume = "truststore-volume"
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
	GetInitContainersTemplates() []v1.Container
	GetContainerVolumeMounts() []v1.VolumeMount
	GetVolumes() []v1.Volume
}

type capabilityBase struct {
	moduleName     string
	capabilityName string
	properties     *dynatracev1alpha1.CapabilityProperties
	Configuration
	initContainersTemplates []v1.Container
	containerVolumeMounts   []v1.VolumeMount
	volumes                 []v1.Volume
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

// Note:
// Caller must set following fields:
//   Image:
//   Resources:
func (c *capabilityBase) GetInitContainersTemplates() []v1.Container {
	return c.initContainersTemplates
}

func (c *capabilityBase) GetContainerVolumeMounts() []v1.VolumeMount {
	return c.containerVolumeMounts
}

func (c *capabilityBase) GetVolumes() []v1.Volume {
	return c.volumes
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

type DataIngestCapability struct {
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
			initContainersTemplates: []v1.Container{
				{
					Name:            "certificate-loader",
					ImagePullPolicy: v1.PullAlways,
					WorkingDir:      "/var/lib/dynatrace/gateway",
					Command:         []string{"/bin/bash"},
					Args:            []string{"-c", "/opt/dynatrace/gateway/k8scrt2jks.sh"},
					VolumeMounts: []v1.VolumeMount{
						{
							ReadOnly:  false,
							Name:      "truststore-volume",
							MountPath: "/var/lib/dynatrace/gateway/ssl",
						},
					},
				},
			},
			containerVolumeMounts: []v1.VolumeMount{{
				ReadOnly:  true,
				Name:      trustStoreVolume,
				MountPath: "/opt/dynatrace/gateway/jre/lib/security/cacerts",
				SubPath:   "k8s-local.jks",
			}},
			volumes: []v1.Volume{{
				Name: trustStoreVolume,
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			}},
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

func NewDataIngestCapability(crProperties *dynatracev1alpha1.CapabilityProperties) *DataIngestCapability {
	return &DataIngestCapability{
		capabilityBase{
			moduleName:     "data-ingest",
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
