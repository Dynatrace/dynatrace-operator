package activegate

import (
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
)

const (
	TenantSecretSuffix            = "-activegate-tenant-secret"
	ConnectionInfoConfigMapSuffix = "-activegate-connection-info"
	AuthTokenSecretSuffix         = "-activegate-authtoken-secret"
	DefaultImageRegistrySubPath   = "/linux/activegate"
)

func (ag *Spec) SetApiUrl(apiUrl string) {
	ag.apiUrl = apiUrl
}

func (ag *Spec) SetName(name string) {
	ag.name = name
}

func (ag *Spec) SetExtensionsDependency(isEnabled bool) {
	ag.enabledDependencies.extensions = isEnabled
}

func (ag *Spec) apiUrlHost() string {
	parsedUrl, err := url.Parse(ag.apiUrl)
	if err != nil {
		return ""
	}

	return parsedUrl.Host
}

// NeedsActiveGate returns true when a feature requires ActiveGate instances.
func (ag *Spec) IsEnabled() bool {
	return len(ag.Capabilities) > 0 || ag.enabledDependencies.Any()
}

func (ag *Spec) IsMode(mode CapabilityDisplayName) bool {
	for _, capability := range ag.Capabilities {
		if capability == mode {
			return true
		}
	}

	return false
}

func (ag *Spec) GetServiceAccountOwner() string {
	if ag.IsKubernetesMonitoringEnabled() {
		return string(KubeMonCapability.DisplayName)
	} else {
		return "activegate"
	}
}

func (ag *Spec) GetServiceAccountName() string {
	return "dynatrace-" + ag.GetServiceAccountOwner()
}

func (ag *Spec) IsKubernetesMonitoringEnabled() bool {
	return ag.IsMode(KubeMonCapability.DisplayName)
}

func (ag *Spec) IsRoutingEnabled() bool {
	return ag.IsMode(RoutingCapability.DisplayName)
}

func (ag *Spec) IsApiEnabled() bool {
	return ag.IsMode(DynatraceApiCapability.DisplayName)
}

func (ag *Spec) IsMetricsIngestEnabled() bool {
	return ag.IsMode(MetricsIngestCapability.DisplayName)
}

func (ag *Spec) NeedsService() bool {
	return ag.IsRoutingEnabled() ||
		ag.IsApiEnabled() ||
		ag.IsMetricsIngestEnabled() ||
		ag.enabledDependencies.extensions
}

func (ag *Spec) HasCaCert() bool {
	return ag.IsEnabled() && ag.TlsSecretName != ""
}

// ActivegateTenantSecret returns the name of the secret containing tenant UUID, token and communication endpoints for ActiveGate.
func (ag *Spec) GetTenantSecretName() string {
	return ag.name + TenantSecretSuffix
}

// ActiveGateAuthTokenSecret returns the name of the secret containing the ActiveGateAuthToken, which is mounted to the AGs.
func (ag *Spec) GetAuthTokenSecretName() string {
	return ag.name + AuthTokenSecretSuffix
}

func (ag *Spec) GetConnectionInfoConfigMapName() string {
	return ag.name + ConnectionInfoConfigMapSuffix
}

// DefaultImage provides the image reference for the ActiveGate from tenant registry.
// Format: repo:tag.
func (ag *Spec) GetDefaultImage(version string) string {
	apiUrlHost := ag.apiUrlHost()
	if apiUrlHost == "" {
		return ""
	}

	truncatedVersion := dtversion.ToImageTag(version)
	tag := truncatedVersion

	if !strings.Contains(tag, api.RawTag) {
		tag += "-" + api.RawTag
	}

	return apiUrlHost + DefaultImageRegistrySubPath + ":" + tag
}

// CustomActiveGateImage provides the image reference for the ActiveGate provided in the Spec.
func (ag *Spec) GetCustomImage() string {
	return ag.Image
}
