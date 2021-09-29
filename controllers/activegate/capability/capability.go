package capability

import (

	//	"path/filepath"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const (
	trustStoreVolume          = "truststore-volume"
	k8scrt2jksPath            = "/opt/dynatrace/gateway/k8scrt2jks.sh"
	activeGateCacertsPath     = "/opt/dynatrace/gateway/jre/lib/security/cacerts"
	activeGateSslPath         = "/var/lib/dynatrace/gateway/ssl"
	k8sCertificateFile        = "k8s-local.jks"
	k8scrt2jksWorkingDir      = "/var/lib/dynatrace/gateway"
	initContainerTemplateName = "certificate-loader"

	//jettyCerts = "server-certs"

	//secretsRootDir = "/var/lib/dynatrace/secrets/"
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
	GetProperties() *dynatracev1beta1.CapabilityProperties
	GetConfiguration() Configuration
	GetInitContainersTemplates() []corev1.Container
	GetContainerVolumeMounts() []corev1.VolumeMount
	GetVolumes() []corev1.Volume
}

type capabilityBase struct {
	moduleName     string
	capabilityName string
	properties     *dynatracev1beta1.CapabilityProperties
	Configuration
	initContainersTemplates []corev1.Container
	containerVolumeMounts   []corev1.VolumeMount
	volumes                 []corev1.Volume
}

func (c *capabilityBase) GetProperties() *dynatracev1beta1.CapabilityProperties {
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
func (c *capabilityBase) GetInitContainersTemplates() []corev1.Container {
	return c.initContainersTemplates
}

func (c *capabilityBase) GetContainerVolumeMounts() []corev1.VolumeMount {
	return c.containerVolumeMounts
}

func (c *capabilityBase) GetVolumes() []corev1.Volume {
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

// func (c *capabilityBase) setTlsConfig(agSpec *dynatracev1beta1.ActiveGateSpec) {
// 	if agSpec == nil {
// 		return
// 	}

// 	if agSpec.TlsSecretName != "" {
// 		c.volumes = append(c.volumes,
// 			corev1.Volume{
// 				Name: jettyCerts,
// 				VolumeSource: corev1.VolumeSource{
// 					Secret: &corev1.SecretVolumeSource{
// 						SecretName: agSpec.TlsSecretName,
// 					},
// 				},
// 			})
// 		c.containerVolumeMounts = append(c.containerVolumeMounts,
// 			corev1.VolumeMount{
// 				ReadOnly:  true,
// 				Name:      jettyCerts,
// 				MountPath: filepath.Join(secretsRootDir, "tls"),
// 			})
// 	}
// }

func NewKubeMonCapability(crProperties *dynatracev1beta1.CapabilityProperties) *KubeMonCapability {
	c := &KubeMonCapability{
		capabilityBase{
			moduleName:     "kubemon",
			capabilityName: "kubernetes_monitoring",
			properties:     crProperties,
			Configuration: Configuration{
				ServiceAccountOwner: "kubernetes-monitoring",
			},
			initContainersTemplates: []corev1.Container{
				{
					Name:            initContainerTemplateName,
					ImagePullPolicy: corev1.PullAlways,
					WorkingDir:      k8scrt2jksWorkingDir,
					Command:         []string{"/bin/bash"},
					Args:            []string{"-c", k8scrt2jksPath},
					VolumeMounts: []corev1.VolumeMount{
						{
							ReadOnly:  false,
							Name:      trustStoreVolume,
							MountPath: activeGateSslPath,
						},
					},
				},
			},
			containerVolumeMounts: []corev1.VolumeMount{{
				ReadOnly:  true,
				Name:      trustStoreVolume,
				MountPath: activeGateCacertsPath,
				SubPath:   k8sCertificateFile,
			}},
			volumes: []corev1.Volume{{
				Name: trustStoreVolume,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				}},
			},
		},
	}
	return c
}

func NewRoutingCapability(crProperties *dynatracev1beta1.CapabilityProperties) *RoutingCapability {
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
	return c
}

func NewDataIngestCapability(crProperties *dynatracev1beta1.CapabilityProperties) *DataIngestCapability {
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
	return c
}
