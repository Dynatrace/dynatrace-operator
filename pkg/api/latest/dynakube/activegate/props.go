package activegate

import (
	"net/url"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
)

const (
	TenantSecretSuffix            = "-activegate-tenant-secret"
	TLSSecretSuffix               = "-activegate-tls-secret"
	ConnectionInfoConfigMapSuffix = "-activegate-connection-info"
	AuthTokenSecretSuffix         = "-activegate-authtoken-secret"
	DefaultImageRegistrySubPath   = "/linux/activegate"
)

func (ag *Spec) SetAPIURL(apiURL string) {
	ag.apiURL = apiURL
}

func (ag *Spec) SetName(name string) {
	ag.name = name
}

func (ag *Spec) SetAutomaticTLSCertificate(enabled bool) {
	ag.automaticTLSCertificateEnabled = enabled
}
func (ag *Spec) SetExtensionsDependency(isEnabled bool) {
	ag.enabledDependencies.extensions = isEnabled
}

func (ag *Spec) apiURLHost() string {
	parsedURL, err := url.Parse(ag.apiURL)
	if err != nil {
		return ""
	}

	return parsedURL.Host
}

// IsEnabled returns true when a feature requires ActiveGate instances.
func (ag *Spec) IsEnabled() bool {
	return len(ag.Capabilities) > 0 || ag.enabledDependencies.Any()
}

func (ag *Spec) IsMode(mode CapabilityDisplayName) bool {
	return slices.Contains(ag.Capabilities, mode)
}

func (ag *Spec) GetServiceAccountOwner() string {
	if ag.IsKubernetesMonitoringEnabled() {
		return string(KubeMonCapability.DisplayName)
	} else {
		return "activegate"
	}
}

func (ag *Spec) GetReplicas() int32 {
	var defaultReplicas int32 = 1
	if ag.Replicas == nil {
		return defaultReplicas
	}

	return *ag.Replicas
}

func (ag *Spec) GetServiceAccountName() string {
	return "dynatrace-activegate"
}

func (ag *Spec) IsKubernetesMonitoringEnabled() bool {
	return ag.IsMode(KubeMonCapability.DisplayName)
}

func (ag *Spec) IsRoutingEnabled() bool {
	return ag.IsMode(RoutingCapability.DisplayName)
}

func (ag *Spec) IsAPIEnabled() bool {
	return ag.IsMode(DynatraceAPICapability.DisplayName)
}

func (ag *Spec) IsMetricsIngestEnabled() bool {
	return ag.IsMode(MetricsIngestCapability.DisplayName)
}

func (ag *Spec) IsAutomaticTLSSecretEnabled() bool {
	return ag.automaticTLSCertificateEnabled
}

func (ag *Spec) HasCaCert() bool {
	return ag.IsEnabled() && (ag.TLSSecretName != "" || ag.IsAutomaticTLSSecretEnabled())
}

// GetTenantSecretName returns the name of the secret containing tenant UUID, token and communication endpoints for ActiveGate.
func (ag *Spec) GetTenantSecretName() string {
	return ag.name + TenantSecretSuffix
}

// GetAuthTokenSecretName returns the name of the secret containing the ActiveGateAuthToken, which is mounted to the AGs.
func (ag *Spec) GetAuthTokenSecretName() string {
	return ag.name + AuthTokenSecretSuffix
}

// GetTLSSecretName returns the name of the AG TLS secret.
func (ag *Spec) GetTLSSecretName() string {
	if ag.TLSSecretName != "" {
		return ag.TLSSecretName
	}

	if ag.IsAutomaticTLSSecretEnabled() {
		return ag.GetAutoTLSSecretName()
	}

	return ""
}

// GetAutoTLSSecretName returns the name of the automatically created AG TLS secret.
func (ag *Spec) GetAutoTLSSecretName() string {
	return ag.name + TLSSecretSuffix
}

func (ag *Spec) GetConnectionInfoConfigMapName() string {
	return ag.name + ConnectionInfoConfigMapSuffix
}

// GetDefaultImage provides the image reference for the ActiveGate from tenant registry.
// Format: repo:tag.
func (ag *Spec) GetDefaultImage(version string) string {
	apiURLHost := ag.apiURLHost()
	if apiURLHost == "" {
		return ""
	}

	truncatedVersion := dtversion.ToImageTag(version)
	tag := truncatedVersion

	if !strings.Contains(tag, api.RawTag) {
		tag += "-" + api.RawTag
	}

	return apiURLHost + DefaultImageRegistrySubPath + ":" + tag
}

// GetCustomImage provides the image reference for the ActiveGate provided in the Spec.
func (ag *Spec) GetCustomImage() string {
	return ag.Image
}

// GetTerminationGracePeriodSeconds provides the configured value for the terminatGracePeriodSeconds parameter of the pod.
func (ag *Spec) GetTerminationGracePeriodSeconds() *int64 { return ag.TerminationGracePeriodSeconds }
