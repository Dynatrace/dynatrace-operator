package capability

import (
	"path/filepath"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const (
	multiActiveGatePodName    = "activegates"
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

type baseFunc func() *capabilityBase

var activeGateCapabilities = map[dynatracev1beta1.CapabilityDisplayName]baseFunc{
	dynatracev1beta1.KubeMonCapability.DisplayName:    kubeMonBase,
	dynatracev1beta1.RoutingCapability.DisplayName:    routingBase,
	dynatracev1beta1.DataIngestCapability.DisplayName: dataIngestBase,
}

type Configuration struct {
	SetDnsEntryPoint     bool
	SetReadinessPort     bool
	SetCommunicationPort bool
	CreateService        bool
	ServiceAccountOwner  string
}

type Capability interface {
	Enabled() bool
	GetShortName() string
	GetArgName() string
	GetProperties() *dynatracev1beta1.CapabilityProperties
	GetConfiguration() Configuration
	GetInitContainersTemplates() []corev1.Container
	GetContainerVolumeMounts() []corev1.VolumeMount
	GetVolumes() []corev1.Volume
}

type capabilityBase struct {
	enabled    bool
	shortName  string
	ArgName    string
	properties *dynatracev1beta1.CapabilityProperties
	Configuration
	initContainersTemplates []corev1.Container
	containerVolumeMounts   []corev1.VolumeMount
	volumes                 []corev1.Volume
}

func (c *capabilityBase) Enabled() bool {
	return c.enabled
}

func (c *capabilityBase) GetProperties() *dynatracev1beta1.CapabilityProperties {
	return c.properties
}

func (c *capabilityBase) GetConfiguration() Configuration {
	return c.Configuration
}

func (c *capabilityBase) GetShortName() string {
	return c.shortName
}

func (c *capabilityBase) GetArgName() string {
	return c.ArgName
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
	return instanceName + "-" + capability.GetShortName()
}

// Deprecated
type KubeMonCapability struct {
	capabilityBase
}

// Deprecated
type RoutingCapability struct {
	capabilityBase
}

type MultiCapability struct {
	capabilityBase
}

func (c *capabilityBase) setTlsConfig(agSpec *dynatracev1beta1.ActiveGateSpec) {
	if agSpec == nil {
		return
	}

	if agSpec.TlsSecretName != "" {
		c.volumes = append(c.volumes,
			corev1.Volume{
				Name: jettyCerts,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: agSpec.TlsSecretName,
					},
				},
			})
		c.containerVolumeMounts = append(c.containerVolumeMounts,
			corev1.VolumeMount{
				ReadOnly:  true,
				Name:      jettyCerts,
				MountPath: filepath.Join(secretsRootDir, "tls"),
			})
	}
}

func NewMultiCapability(dk *dynatracev1beta1.DynaKube) *MultiCapability {
	mc := MultiCapability{
		capabilityBase{
			shortName: multiActiveGatePodName,
		},
	}
	if dk == nil || !dk.ActiveGateMode() {
		return &mc
	}
	mc.enabled = true
	mc.properties = &dk.Spec.ActiveGate.CapabilityProperties
	capabilityNames := []string{}
	for _, capName := range dk.Spec.ActiveGate.Capabilities {
		cap := activeGateCapabilities[capName]()
		capabilityNames = append(capabilityNames, cap.ArgName)
		mc.initContainersTemplates = append(mc.initContainersTemplates, cap.initContainersTemplates...)
		mc.containerVolumeMounts = append(mc.containerVolumeMounts, cap.containerVolumeMounts...)
		mc.volumes = append(mc.volumes, cap.volumes...)

		if !mc.CreateService {
			mc.CreateService = cap.CreateService
		}
		if !mc.SetCommunicationPort {
			mc.SetCommunicationPort = cap.SetCommunicationPort
		}
		if !mc.SetDnsEntryPoint {
			mc.SetDnsEntryPoint = cap.SetDnsEntryPoint
		}
		if !mc.SetReadinessPort {
			mc.SetReadinessPort = cap.SetReadinessPort
		}
		if mc.ServiceAccountOwner == "" {
			mc.ServiceAccountOwner = cap.ServiceAccountOwner
		}
	}
	mc.ArgName = strings.Join(capabilityNames[:], ",")
	mc.setTlsConfig(&dk.Spec.ActiveGate)
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
		ArgName:   dynatracev1beta1.KubeMonCapability.ArgumentName,
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
	}
	return &c
}

func routingBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.RoutingCapability.ShortName,
		ArgName:   dynatracev1beta1.RoutingCapability.ArgumentName,
		Configuration: Configuration{
			SetDnsEntryPoint:     true,
			SetReadinessPort:     true,
			SetCommunicationPort: true,
			CreateService:        true,
		},
	}
	return &c
}

func dataIngestBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.DataIngestCapability.ShortName,
		ArgName:   dynatracev1beta1.DataIngestCapability.ArgumentName,
		Configuration: Configuration{
			SetDnsEntryPoint:     true,
			SetReadinessPort:     true,
			SetCommunicationPort: true,
			CreateService:        true,
		},
	}
	return &c
}
