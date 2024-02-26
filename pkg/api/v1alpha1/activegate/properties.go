package activegate

import (
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
)

const (
	ActiveGateTenantSecretSuffix            = "-activegate-tenant-secret"
	ActiveGateConnectionInfoConfigMapSuffix = "-activegate-connection-info"
	AuthTokenSecretSuffix                   = "-activegate-authtoken-secret"

	defaultActiveGateImage = "/linux/activegate:raw"
)

// ApiUrl is a getter for activeGate.Spec.Specific.APIURL.
func (activeGate *ActiveGate) ApiUrl() string {
	return activeGate.Spec.Common.APIURL
}

// ApiUrlHost returns the host of activeGate.Spec.Specific.APIURL
// E.g. if the APIURL is set to "https://my-tenant.dynatrace.com/api", it returns "my-tenant.dynatrace.com"
// If the URL cannot be parsed, it returns an empty string.
func (activeGate *ActiveGate) ApiUrlHost() string {
	parsedUrl, err := url.Parse(activeGate.ApiUrl())
	if err != nil {
		return ""
	}

	return parsedUrl.Host
}

func (activeGate *ActiveGate) ActiveGateMode() bool {
	return len(activeGate.Spec.Specific.Capabilities) > 0
}

func (activeGate *ActiveGate) IsActiveGateMode(mode CapabilityDisplayName) bool {
	for _, capability := range activeGate.Spec.Specific.Capabilities {
		if capability == mode {
			return true
		}
	}

	return false
}

func (activeGate *ActiveGate) IsKubernetesMonitoringActiveGateEnabled() bool {
	return activeGate.IsActiveGateMode(KubeMonCapability.DisplayName)
}

func (activeGate *ActiveGate) IsRoutingActiveGateEnabled() bool {
	return activeGate.IsActiveGateMode(RoutingCapability.DisplayName)
}

func (activeGate *ActiveGate) IsApiActiveGateEnabled() bool {
	return activeGate.IsActiveGateMode(DynatraceApiCapability.DisplayName)
}

func (activeGate *ActiveGate) IsMetricsIngestActiveGateEnabled() bool {
	return activeGate.IsActiveGateMode(MetricsIngestCapability.DisplayName)
}

func (activeGate *ActiveGate) NeedsActiveGateServicePorts() bool {
	return activeGate.IsRoutingActiveGateEnabled() ||
		activeGate.IsApiActiveGateEnabled() ||
		activeGate.IsMetricsIngestActiveGateEnabled()
}

func (activeGate *ActiveGate) NeedsActiveGateService() bool {
	return activeGate.NeedsActiveGateServicePorts()
}

func (activeGate *ActiveGate) HasActiveGateCaCert() bool {
	return activeGate.ActiveGateMode() && activeGate.Spec.Specific.TlsSecretName != ""
}

// ActivegateTenantSecret returns the name of the secret containing tenant UUID, token and communication endpoints for ActiveGate.
func (activeGate *ActiveGate) ActivegateTenantSecret() string {
	return activeGate.Name + ActiveGateTenantSecretSuffix
}

// ActiveGateAuthTokenSecret returns the name of the secret containing the ActiveGateAuthToken, which is mounted to the AGs.
func (activeGate *ActiveGate) ActiveGateAuthTokenSecret() string {
	return activeGate.Name + AuthTokenSecretSuffix
}

func (activeGate *ActiveGate) ActiveGateConnectionInfoConfigMapName() string {
	return activeGate.Name + ActiveGateConnectionInfoConfigMapSuffix
}

// ActiveGateImage provides the image reference set in Status for the ActiveGate.
// Format: repo@sha256:digest.
func (activeGate *ActiveGate) ActiveGateImage() string {
	return activeGate.Status.ImageID
}

// DefaultActiveGateImage provides the image reference for the ActiveGate from tenant registry.
// Format: repo:tag.
func (activeGate *ActiveGate) DefaultActiveGateImage() string {
	apiUrlHost := activeGate.ApiUrlHost()

	if apiUrlHost == "" {
		return ""
	}

	return apiUrlHost + defaultActiveGateImage
}

// CustomActiveGateImage provides the image reference for the ActiveGate provided in the Spec.
func (activeGate *ActiveGate) CustomActiveGateImage() string {
	return activeGate.Spec.Specific.Image
}

// Tokens returns the name of the Secret to be used for tokens.
func (activeGate *ActiveGate) Tokens() string {
	if tkns := activeGate.Spec.Common.Tokens; tkns != "" {
		return tkns
	}

	return activeGate.Name
}

// TenantUUIDFromApiUrl gets the tenantUUID from the ApiUrl present in the struct, if the tenant is aliased then the alias will be returned.
func (activeGate *ActiveGate) TenantUUIDFromApiUrl() (string, error) {
	return tenantUUID(activeGate.Spec.Common.APIURL)
}

func runeIs(wanted rune) func(rune) bool {
	return func(actual rune) bool {
		return actual == wanted
	}
}

func tenantUUID(apiUrl string) (string, error) {
	parsedUrl, err := url.Parse(apiUrl)
	if err != nil {
		return "", errors.WithMessagef(err, "problem parsing tenant id from url %s", apiUrl)
	}

	// Path = "/e/<token>/api" -> ["e",  "<tenant>", "api"]
	subPaths := strings.FieldsFunc(parsedUrl.Path, runeIs('/'))
	if len(subPaths) >= 3 && subPaths[0] == "e" && subPaths[2] == "api" {
		return subPaths[1], nil
	}

	hostnameWithDomains := strings.FieldsFunc(parsedUrl.Hostname(), runeIs('.'))
	if len(hostnameWithDomains) >= 1 {
		return hostnameWithDomains[0], nil
	}

	return "", errors.Errorf("problem getting tenant id from API URL '%s'", apiUrl)
}

func (activeGate *ActiveGate) IsActiveGateConnectionInfoUpdateAllowed(timeProvider *timeprovider.Provider) bool {
	return timeProvider.IsOutdated(&activeGate.Status.ConnectionInfoStatus.LastRequest, activeGate.FeatureApiRequestThreshold())
}
