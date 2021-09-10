package capability

import (
	"path/filepath"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

const (
	trustStoreVolume          = "truststore-volume"
	k8scrt2jksPath            = "/opt/dynatrace/gateway/k8scrt2jks.sh"
	activeGateCacertsPath     = "/opt/dynatrace/gateway/jre/lib/security/cacerts"
	activeGateSslPath         = "/var/lib/dynatrace/gateway/ssl"
	k8sCertificateFile        = "k8s-local.jks"
	k8scrt2jksWorkingDir      = "/var/lib/dynatrace/gateway"
	initContainerTemplateName = "certificate-loader"

	jettyCerts = "server-certs"

	secretsRootDir = "/var/lib/dynatrace/secrets/"
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

func (c *capabilityBase) setTlsConfig(agSpec *dynatracev1alpha1.ActiveGateSpec) {
	if agSpec == nil {
		return
	}

	if agSpec.TlsSecretName != "" {
		c.volumes = append(c.volumes,
			v1.Volume{
				Name: jettyCerts,
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: agSpec.TlsSecretName,
					},
				},
			})
		c.containerVolumeMounts = append(c.containerVolumeMounts,
			v1.VolumeMount{
				ReadOnly:  true,
				Name:      jettyCerts,
				MountPath: filepath.Join(secretsRootDir, "tls"),
			})
	}
}

func NewKubeMonCapability(crProperties *dynatracev1alpha1.CapabilityProperties, agSpec *dynatracev1alpha1.ActiveGateSpec) *KubeMonCapability {
	c := &KubeMonCapability{
		capabilityBase{
			moduleName:     "kubemon",
			capabilityName: "kubernetes_monitoring",
			properties:     crProperties,
			Configuration: Configuration{
				ServiceAccountOwner: "kubernetes-monitoring",
			},
			initContainersTemplates: []v1.Container{
				{
					Name:            initContainerTemplateName,
					ImagePullPolicy: v1.PullAlways,
					WorkingDir:      k8scrt2jksWorkingDir,
					Command:         []string{"/bin/bash"},
					Args:            []string{"-c", k8scrt2jksPath},
					VolumeMounts: []v1.VolumeMount{
						{
							ReadOnly:  false,
							Name:      trustStoreVolume,
							MountPath: activeGateSslPath,
						},
					},
				},
			},
			containerVolumeMounts: []v1.VolumeMount{{
				ReadOnly:  true,
				Name:      trustStoreVolume,
				MountPath: activeGateCacertsPath,
				SubPath:   k8sCertificateFile,
			}},
			volumes: []v1.Volume{{
				Name: trustStoreVolume,
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				}},
			},
		},
	}

	c.capabilityBase.setTlsConfig(agSpec)

	return c
}

func NewRoutingCapability(crProperties *dynatracev1alpha1.CapabilityProperties, agSpec *dynatracev1alpha1.ActiveGateSpec) *RoutingCapability {
	c := &RoutingCapability{
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

	c.capabilityBase.setTlsConfig(agSpec)

	return c
}

func NewDataIngestCapability(crProperties *dynatracev1alpha1.CapabilityProperties, agSpec *dynatracev1alpha1.ActiveGateSpec) *DataIngestCapability {
	c := &DataIngestCapability{
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

	c.capabilityBase.setTlsConfig(agSpec)

	return c
}
