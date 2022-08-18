package capability

import (
	"path/filepath"
	"regexp"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
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

	jettyCerts = "server-certs"

	secretsRootDir = "/var/lib/dynatrace/secrets/"
)

type baseFunc func() *capabilityBase

var activeGateCapabilities = map[dynatracev1beta1.CapabilityDisplayName]baseFunc{
	dynatracev1beta1.KubeMonCapability.DisplayName:       kubeMonBase,
	dynatracev1beta1.RoutingCapability.DisplayName:       routingBase,
	dynatracev1beta1.MetricsIngestCapability.DisplayName: metricsIngestBase,
	dynatracev1beta1.DynatraceApiCapability.DisplayName:  dynatraceApiBase,
	dynatracev1beta1.StatsdIngestCapability.DisplayName:  statsdIngestBase,
}

type AgServicePorts struct {
	Webserver bool
	Statsd    bool
}

func (ports AgServicePorts) HasPorts() bool {
	return ports.Webserver || ports.Statsd
}

type Configuration struct {
	SetDnsEntryPoint       bool
	SetReadinessPort       bool
	SetCommunicationPort   bool
	ServicePorts           AgServicePorts
	CreateEecRuntimeConfig bool
	ServiceAccountOwner    string
}

type Capability interface {
	Enabled() bool
	ShortName() string
	ArgName() string
	Properties() *dynatracev1beta1.CapabilityProperties
	Config() Configuration
	InitContainersTemplates() []corev1.Container
	ContainerVolumeMounts() []corev1.VolumeMount
	Volumes() []corev1.Volume
	ShouldCreateService() bool
}

type capabilityBase struct {
	enabled    bool
	shortName  string
	argName    string
	properties *dynatracev1beta1.CapabilityProperties
	Configuration
	initContainersTemplates []corev1.Container
	containerVolumeMounts   []corev1.VolumeMount
	volumes                 []corev1.Volume
}

func (c *capabilityBase) Enabled() bool {
	return c.enabled
}

func (c *capabilityBase) Properties() *dynatracev1beta1.CapabilityProperties {
	return c.properties
}

func (c *capabilityBase) Config() Configuration {
	return c.Configuration
}

func (c *capabilityBase) ShortName() string {
	return c.shortName
}

func (c *capabilityBase) ArgName() string {
	return c.argName
}

func (c *capabilityBase) ShouldCreateService() bool {
	return c.ServicePorts.HasPorts()
}

// Note:
// Caller must set following fields:
//
//	Image:
//	Resources:
func (c *capabilityBase) InitContainersTemplates() []corev1.Container {
	return c.initContainersTemplates
}

func (c *capabilityBase) ContainerVolumeMounts() []corev1.VolumeMount {
	return c.containerVolumeMounts
}

func (c *capabilityBase) Volumes() []corev1.Volume {
	return c.volumes
}

func CalculateStatefulSetName(capability Capability, dynakubeName string) string {
	return dynakubeName + "-" + capability.ShortName()
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
			shortName: consts.MultiActiveGateName,
		},
	}
	if dk == nil || !dk.ActiveGateMode() {
		mc.ServicePorts.Webserver = true // necessary for cleaning up service if created
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
		mc.initContainersTemplates = append(mc.initContainersTemplates, capGen.initContainersTemplates...)
		mc.containerVolumeMounts = append(mc.containerVolumeMounts, capGen.containerVolumeMounts...)
		mc.volumes = append(mc.volumes, capGen.volumes...)

		if !mc.ServicePorts.Webserver {
			mc.ServicePorts.Webserver = capGen.ServicePorts.Webserver
		}
		if !mc.ServicePorts.Statsd {
			mc.ServicePorts.Statsd = capGen.ServicePorts.Statsd
		}
		if !mc.CreateEecRuntimeConfig {
			mc.CreateEecRuntimeConfig = capGen.CreateEecRuntimeConfig
		}
		if !mc.SetCommunicationPort {
			mc.SetCommunicationPort = capGen.SetCommunicationPort
		}
		if !mc.SetDnsEntryPoint {
			mc.SetDnsEntryPoint = capGen.SetDnsEntryPoint
		}
		if !mc.SetReadinessPort {
			mc.SetReadinessPort = capGen.SetReadinessPort
		}
		if mc.ServiceAccountOwner == "" {
			mc.ServiceAccountOwner = capGen.ServiceAccountOwner
		}
	}
	mc.argName = strings.Join(capabilityNames, ",")
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
		argName:   dynatracev1beta1.KubeMonCapability.ArgumentName,
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
		argName:   dynatracev1beta1.RoutingCapability.ArgumentName,
		Configuration: Configuration{
			SetDnsEntryPoint:     true,
			SetReadinessPort:     true,
			SetCommunicationPort: true,
			ServicePorts: AgServicePorts{
				Webserver: true,
			},
			CreateEecRuntimeConfig: false,
		},
	}
	return &c
}

func metricsIngestBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.MetricsIngestCapability.ShortName,
		argName:   dynatracev1beta1.MetricsIngestCapability.ArgumentName,
		Configuration: Configuration{
			SetDnsEntryPoint:     true,
			SetReadinessPort:     true,
			SetCommunicationPort: true,
			ServicePorts: AgServicePorts{
				Webserver: true,
			},
			CreateEecRuntimeConfig: false,
		},
	}
	return &c
}

func dynatraceApiBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.DynatraceApiCapability.ShortName,
		argName:   dynatracev1beta1.DynatraceApiCapability.ArgumentName,
		Configuration: Configuration{
			SetDnsEntryPoint:     true,
			SetReadinessPort:     true,
			SetCommunicationPort: true,
			ServicePorts: AgServicePorts{
				Webserver: true,
			},
		},
	}
	return &c
}

func statsdIngestBase() *capabilityBase {
	c := capabilityBase{
		shortName: dynatracev1beta1.StatsdIngestCapability.ShortName,
		argName:   dynatracev1beta1.StatsdIngestCapability.ArgumentName,
		Configuration: Configuration{
			SetDnsEntryPoint:     true,
			SetReadinessPort:     true,
			SetCommunicationPort: true,
			ServicePorts: AgServicePorts{
				Statsd: true,
			},
			CreateEecRuntimeConfig: true,
		},
	}
	return &c
}

func GenerateActiveGateCapabilities(dynakube *dynatracev1beta1.DynaKube) []Capability {
	return []Capability{
		NewKubeMonCapability(dynakube),
		NewRoutingCapability(dynakube),
		NewMultiCapability(dynakube),
	}
}

func BuildEecConfigMapName(dynakubeName string, module string) string {
	return regexp.MustCompile(`[^\w\-]`).ReplaceAllString(dynakubeName+"-"+module+"-eec-config", "_")
}

func BuildProxySecretName() string {
	return "dynatrace" + "-" + consts.MultiActiveGateName + "-" + consts.ProxySecretSuffix
}
