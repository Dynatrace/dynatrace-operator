// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package activegate

import (
	"net/url"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
)

const (
	TenantSecretSuffix                  = "-activegate-tenant-secret"
	TLSSecretSuffix                     = "-activegate-tls-secret"
	ConnectionInfoConfigMapSuffix       = "-activegate-connection-info"
	DeploymentPropertiesConfigMapSuffix = "-activegate-deployment-properties"
	AuthTokenSecretSuffix               = "-activegate-authtoken-secret"
	DefaultImageRegistrySubPath         = "/linux/activegate"
)

func (ag *ActiveGate) SetAPIURL(apiURL string) {
	ag.apiURL = apiURL
}

func (ag *ActiveGate) SetName(name string) {
	ag.name = name
}

func (ag *ActiveGate) SetAutomaticTLSCertificate(enabled bool) {
	ag.automaticTLSCertificateEnabled = enabled
}
func (ag *ActiveGate) SetExtensionsDependency(isEnabled bool) {
	ag.enabledDependencies.extensions = isEnabled
}

func (ag *ActiveGate) apiURLHost() string {
	parsedURL, err := url.Parse(ag.apiURL)
	if err != nil {
		return ""
	}

	return parsedURL.Host
}

// IsEnabled returns true when a feature requires ActiveGate instances.
func (ag *ActiveGate) IsEnabled() bool {
	return len(ag.Capabilities) > 0 || ag.enabledDependencies.Any()
}

func (ag *ActiveGate) IsMode(mode CapabilityDisplayName) bool {
	return slices.Contains(ag.Capabilities, mode)
}

func (ag *ActiveGate) GetServiceAccountOwner() string {
	if ag.IsKubernetesMonitoringEnabled() {
		return string(KubeMonCapability.DisplayName)
	} else {
		return "activegate"
	}
}

func (ag *ActiveGate) GetReplicas() int32 {
	var defaultReplicas int32 = 1
	if ag.Replicas == nil {
		return defaultReplicas
	}

	return *ag.Replicas
}

func (ag *ActiveGate) GetServiceAccountName() string {
	return "dynatrace-activegate"
}

func (ag *ActiveGate) IsKubernetesMonitoringEnabled() bool {
	return ag.IsMode(KubeMonCapability.DisplayName)
}

func (ag *ActiveGate) IsRoutingEnabled() bool {
	return ag.IsMode(RoutingCapability.DisplayName)
}

func (ag *ActiveGate) IsAPIEnabled() bool {
	return ag.IsMode(DynatraceAPICapability.DisplayName)
}

func (ag *ActiveGate) IsMetricsIngestEnabled() bool {
	return ag.IsMode(MetricsIngestCapability.DisplayName)
}

func (ag *ActiveGate) IsAutomaticTLSSecretEnabled() bool {
	return ag.automaticTLSCertificateEnabled
}

func (ag *ActiveGate) HasCaCert() bool {
	return ag.IsEnabled() && (ag.IsAutomaticTLSSecretEnabled() || ag.TLSSecretName != "")
}

// GetTenantSecretName returns the name of the secret containing tenant UUID, token and communication endpoints for ActiveGate.
func (ag *ActiveGate) GetTenantSecretName() string {
	return ag.name + TenantSecretSuffix
}

// GetAuthTokenSecretName returns the name of the secret containing the ActiveGateAuthToken, which is mounted to the AGs.
func (ag *ActiveGate) GetAuthTokenSecretName() string {
	return ag.name + AuthTokenSecretSuffix
}

// GetTLSSecretName returns the name of the AG TLS secret.
func (ag *ActiveGate) GetTLSSecretName() string {
	if ag.TLSSecretName != "" {
		return ag.TLSSecretName
	}

	if ag.IsAutomaticTLSSecretEnabled() {
		return ag.GetAutoTLSSecretName()
	}

	return ""
}

// GetAutoTLSSecretName returns the name of the automatically created AG TLS secret.
func (ag *ActiveGate) GetCustomTLSSecretName() string {
	if ag.TLSSecretName != "" {
		return ag.TLSSecretName
	}

	return ""
}

// GetAutoTLSSecretName returns the name of the automatically created AG TLS secret.
func (ag *ActiveGate) GetAutoTLSSecretName() string {
	return ag.name + TLSSecretSuffix
}

func (ag *ActiveGate) GetConnectionInfoConfigMapName() string {
	return ag.name + ConnectionInfoConfigMapSuffix
}

func (ag *ActiveGate) GetDeploymentPropertiesConfigMapName() string {
	return ag.name + DeploymentPropertiesConfigMapSuffix
}

// GetDefaultImage provides the image reference for the ActiveGate from tenant registry.
// Format: repo:tag.
func (ag *ActiveGate) GetDefaultImage(version string) string {
	apiURLHost := ag.apiURLHost()
	if apiURLHost == "" {
		return ""
	}

	truncatedVersion := dtversion.ToImageTag(version)
	tag := truncatedVersion

	if !strings.HasSuffix(tag, api.RawTag) {
		tag += "-" + api.RawTag
	}

	return apiURLHost + DefaultImageRegistrySubPath + ":" + tag
}

// GetCustomImage provides the image reference for the ActiveGate provided in the Spec.
func (ag *ActiveGate) GetCustomImage() string {
	return ag.Image
}

// GetTerminationGracePeriodSeconds provides the configured value for the terminatGracePeriodSeconds parameter of the pod.
func (ag *ActiveGate) GetTerminationGracePeriodSeconds() *int64 {
	return ag.TerminationGracePeriodSeconds
}
