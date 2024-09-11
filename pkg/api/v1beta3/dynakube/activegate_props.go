package dynakube

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
)

const (
	ActiveGateTenantSecretSuffix            = "-activegate-tenant-secret"
	ActiveGateConnectionInfoConfigMapSuffix = "-activegate-connection-info"
	AuthTokenSecretSuffix                   = "-activegate-authtoken-secret"
	DefaultActiveGateImageRegistrySubPath   = "/linux/activegate"
)

// NeedsActiveGate returns true when a feature requires ActiveGate instances.
func (dk *DynaKube) NeedsActiveGate() bool {
	return dk.ActiveGateMode()
}

func (dk *DynaKube) ActiveGateMode() bool {
	return len(dk.Spec.ActiveGate.Capabilities) > 0 || dk.IsExtensionsEnabled()
}

func (dk *DynaKube) IsActiveGateMode(mode CapabilityDisplayName) bool {
	for _, capability := range dk.Spec.ActiveGate.Capabilities {
		if capability == mode {
			return true
		}
	}

	return false
}

func (dk *DynaKube) ActiveGateServiceAccountOwner() string {
	if dk.IsKubernetesMonitoringActiveGateEnabled() {
		return string(KubeMonCapability.DeepCopy().DisplayName)
	} else {
		return "activegate"
	}
}

func (dk *DynaKube) ActiveGateServiceAccountName() string {
	return "dynatrace-" + dk.ActiveGateServiceAccountOwner()
}

func (dk *DynaKube) IsKubernetesMonitoringActiveGateEnabled() bool {
	return dk.IsActiveGateMode(KubeMonCapability.DisplayName)
}

func (dk *DynaKube) IsRoutingActiveGateEnabled() bool {
	return dk.IsActiveGateMode(RoutingCapability.DisplayName)
}

func (dk *DynaKube) IsApiActiveGateEnabled() bool {
	return dk.IsActiveGateMode(DynatraceApiCapability.DisplayName)
}

func (dk *DynaKube) IsMetricsIngestActiveGateEnabled() bool {
	return dk.IsActiveGateMode(MetricsIngestCapability.DisplayName)
}

func (dk *DynaKube) NeedsActiveGateService() bool {
	return dk.IsRoutingActiveGateEnabled() ||
		dk.IsApiActiveGateEnabled() ||
		dk.IsMetricsIngestActiveGateEnabled() ||
		dk.IsExtensionsEnabled()
}

func (dk *DynaKube) HasActiveGateCaCert() bool {
	return dk.ActiveGateMode() && dk.Spec.ActiveGate.TlsSecretName != ""
}

// ActivegateTenantSecret returns the name of the secret containing tenant UUID, token and communication endpoints for ActiveGate.
func (dk *DynaKube) ActivegateTenantSecret() string {
	return dk.Name + ActiveGateTenantSecretSuffix
}

// ActiveGateAuthTokenSecret returns the name of the secret containing the ActiveGateAuthToken, which is mounted to the AGs.
func (dk *DynaKube) ActiveGateAuthTokenSecret() string {
	return dk.Name + AuthTokenSecretSuffix
}

func (dk *DynaKube) ActiveGateConnectionInfoConfigMapName() string {
	return dk.Name + ActiveGateConnectionInfoConfigMapSuffix
}

// ActiveGateImage provides the image reference set in Status for the ActiveGate.
// Format: repo@sha256:digest.
func (dk *DynaKube) ActiveGateImage() string {
	return dk.Status.ActiveGate.ImageID
}

// ActiveGateVersion provides version set in Status for the ActiveGate.
func (dk *DynaKube) ActiveGateVersion() string {
	return dk.Status.ActiveGate.Version
}

// DefaultActiveGateImage provides the image reference for the ActiveGate from tenant registry.
// Format: repo:tag.
func (dk *DynaKube) DefaultActiveGateImage(version string) string {
	apiUrlHost := dk.ApiUrlHost()
	if apiUrlHost == "" {
		return ""
	}

	truncatedVersion := dtversion.ToImageTag(version)
	tag := truncatedVersion

	if !strings.Contains(tag, api.RawTag) {
		tag += "-" + api.RawTag
	}

	return apiUrlHost + DefaultActiveGateImageRegistrySubPath + ":" + tag
}

// CustomActiveGateImage provides the image reference for the ActiveGate provided in the Spec.
func (dk *DynaKube) CustomActiveGateImage() string {
	return dk.Spec.ActiveGate.Image
}
