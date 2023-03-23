/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/api"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PullSecretSuffix is the suffix appended to the DynaKube name to n.
	PullSecretSuffix                        = "-pull-secret"
	ActiveGateTenantSecretSuffix            = "-activegate-tenant-secret"
	OneAgentTenantSecretSuffix              = "-oneagent-tenant-secret"
	OneAgentConnectionInfoConfigMapSuffix   = "-oneagent-connection-info"
	ActiveGateConnectionInfoConfigMapSuffix = "-activegate-connection-info"
	AuthTokenSecretSuffix                   = "-activegate-authtoken-secret"
	PodNameOsAgent                          = "oneagent"

	defaultActiveGateImage = "/linux/activegate:latest"
	defaultSyntheticImage  = "linux/dynatrace-synthetic"
)

// ApiUrl is a getter for dk.Spec.APIURL
func (dk *DynaKube) ApiUrl() string {
	return dk.Spec.APIURL
}

// ApiUrlHost returns the host of dk.Spec.APIURL
// E.g. if the APIURL is set to "https://my-tenant.dynatrace.com/api", it returns "my-tenant.dynatrace.com"
// If the URL cannot be parsed, it returns an empty string
func (dk *DynaKube) ApiUrlHost() string {
	parsedUrl, err := url.Parse(dk.ApiUrl())

	if err != nil {
		return ""
	}

	return parsedUrl.Host
}

// NeedsActiveGate returns true when a feature requires ActiveGate instances.
func (dk *DynaKube) NeedsActiveGate() bool {
	return dk.DeprecatedActiveGateMode() ||
		dk.ActiveGateMode() ||
		dk.IsSyntheticMonitoringEnabled()
}

// ApplicationMonitoringMode returns true when application only section is used.
func (dk *DynaKube) ApplicationMonitoringMode() bool {
	return dk.Spec.OneAgent != OneAgentSpec{} && dk.Spec.OneAgent.ApplicationMonitoring != nil
}

// CloudNativeFullstackMode returns true when cloud native fullstack section is used.
func (dk *DynaKube) CloudNativeFullstackMode() bool {
	return dk.Spec.OneAgent != OneAgentSpec{} && dk.Spec.OneAgent.CloudNativeFullStack != nil
}

// HostMonitoringMode returns true when host monitoring section is used.
func (dk *DynaKube) HostMonitoringMode() bool {
	return dk.Spec.OneAgent != OneAgentSpec{} && dk.Spec.OneAgent.HostMonitoring != nil
}

// ClassicFullStackMode returns true when host monitoring section is used.
func (dk *DynaKube) ClassicFullStackMode() bool {
	return dk.Spec.OneAgent != OneAgentSpec{} && dk.Spec.OneAgent.ClassicFullStack != nil
}

// NeedsOneAgent returns true when a feature requires OneAgent instances.
func (dk *DynaKube) NeedsOneAgent() bool {
	return dk.ClassicFullStackMode() || dk.CloudNativeFullstackMode() || dk.HostMonitoringMode()
}

func (dk *DynaKube) OneAgentDaemonsetName() string {
	return fmt.Sprintf("%s-%s", dk.Name, PodNameOsAgent)
}

func (dk *DynaKube) DeprecatedActiveGateMode() bool {
	return dk.Spec.KubernetesMonitoring.Enabled || dk.Spec.Routing.Enabled
}

func (dk *DynaKube) ActiveGateMode() bool {
	return len(dk.Spec.ActiveGate.Capabilities) > 0
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
	return dk.IsActiveGateMode(KubeMonCapability.DisplayName) || dk.Spec.KubernetesMonitoring.Enabled
}

func (dk *DynaKube) IsRoutingActiveGateEnabled() bool {
	return dk.IsActiveGateMode(RoutingCapability.DisplayName) || dk.Spec.Routing.Enabled
}

func (dk *DynaKube) IsApiActiveGateEnabled() bool {
	return dk.IsActiveGateMode(DynatraceApiCapability.DisplayName)
}

func (dk *DynaKube) IsMetricsIngestActiveGateEnabled() bool {
	return dk.IsActiveGateMode(MetricsIngestCapability.DisplayName)
}

func (dk *DynaKube) NeedsActiveGateServicePorts() bool {
	return dk.IsRoutingActiveGateEnabled() ||
		dk.IsApiActiveGateEnabled() ||
		dk.IsMetricsIngestActiveGateEnabled()
}

func (dk *DynaKube) NeedsActiveGateService() bool {
	return dk.NeedsActiveGateServicePorts()
}

func (dynaKube *DynaKube) IsSyntheticMonitoringEnabled() bool {
	return dynaKube.FeatureSyntheticLocationEntityId() != ""
}

func (dk *DynaKube) HasActiveGateCaCert() bool {
	return dk.ActiveGateMode() && dk.Spec.ActiveGate.TlsSecretName != ""
}

func (dk *DynaKube) NeedsOneAgentPrivileged() bool {
	return dk.FeatureOneAgentPrivileged()
}

// ShouldAutoUpdateOneAgent returns true if the Operator should update OneAgent instances automatically.
func (dk *DynaKube) ShouldAutoUpdateOneAgent() bool {
	switch {
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.AutoUpdate == nil || *dk.Spec.OneAgent.CloudNativeFullStack.AutoUpdate
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.AutoUpdate == nil || *dk.Spec.OneAgent.HostMonitoring.AutoUpdate
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.AutoUpdate == nil || *dk.Spec.OneAgent.ClassicFullStack.AutoUpdate
	default:
		return false
	}
}

// ActivegateTenantSecret returns the name of the secret containing tenant UUID, token and communication endpoints for ActiveGate
func (dk *DynaKube) ActivegateTenantSecret() string {
	return dk.Name + ActiveGateTenantSecretSuffix
}

// OneagentTenantSecret returns the name of the secret containing the token for the OneAgent
func (dk *DynaKube) OneagentTenantSecret() string {
	return dk.Name + OneAgentTenantSecretSuffix
}

// ActiveGateAuthTokenSecret returns the name of the secret containing the ActiveGateAuthToken, which is mounted to the AGs
func (dk *DynaKube) ActiveGateAuthTokenSecret() string {
	return dk.Name + AuthTokenSecretSuffix
}

func (dk *DynaKube) ActiveGateConnectionInfoConfigMapName() string {
	return dk.Name + ActiveGateConnectionInfoConfigMapSuffix
}

func (dk *DynaKube) OneAgentConnectionInfoConfigMapName() string {
	return dk.Name + OneAgentConnectionInfoConfigMapSuffix
}

// PullSecret returns the name of the pull secret to be used for immutable images.
func (dk *DynaKube) PullSecret() string {
	if dk.Spec.CustomPullSecret != "" {
		return dk.Spec.CustomPullSecret
	}
	return dk.Name + PullSecretSuffix
}

// ActiveGateImage returns the ActiveGate image to be used with the dk DynaKube instance.
func (dk *DynaKube) ActiveGateImage() string {
	if dk.CustomActiveGateImage() != "" {
		return dk.CustomActiveGateImage()
	}

	return dk.DefaultActiveGateImage()
}

func (dk *DynaKube) DefaultActiveGateImage() string {
	apiUrlHost := dk.ApiUrlHost()

	if apiUrlHost == "" {
		return ""
	}

	return apiUrlHost + defaultActiveGateImage
}

func (dk *DynaKube) deprecatedActiveGateImage() string {
	if dk.Spec.KubernetesMonitoring.Image != "" {
		return dk.Spec.KubernetesMonitoring.Image
	} else if dk.Spec.Routing.Image != "" {
		return dk.Spec.Routing.Image
	}

	return ""
}

func (dk *DynaKube) CustomActiveGateImage() string {
	if dk.DeprecatedActiveGateMode() {
		return dk.deprecatedActiveGateImage()
	}

	return dk.Spec.ActiveGate.Image
}

// returns the synthetic image supplied by the given DynaKube.
func (dk *DynaKube) SyntheticImage() string {
	image := dk.CustomSyntheticImage()
	if image != "" {
		return image
	}
	return dk.DefaultSyntheticImage()
}

func (dk *DynaKube) CustomSyntheticImage() string {
	return dk.FeatureCustomSyntheticImage()
}

func (dk *DynaKube) DefaultSyntheticImage() string {
	if dk.ApiUrl() == "" {
		return ""
	}

	apiUrlHost := dk.ApiUrlHost()
	if apiUrlHost == "" {
		return ""
	}

	return fmt.Sprintf("%s/%s:%s",
		apiUrlHost,
		defaultSyntheticImage,
		api.LatestTag)
}

func (dk *DynaKube) NeedsReadOnlyOneAgents() bool {
	inSupportedMode := dk.HostMonitoringMode() || dk.CloudNativeFullstackMode()
	return inSupportedMode && !dk.FeatureDisableReadOnlyOneAgent()
}

func (dk *DynaKube) NeedsCSIDriver() bool {
	isAppMonitoringWithCSI := dk.ApplicationMonitoringMode() &&
		dk.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver != nil &&
		*dk.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver

	isReadOnlyHostMonitoring := dk.HostMonitoringMode() &&
		!dk.FeatureDisableReadOnlyOneAgent()

	return dk.CloudNativeFullstackMode() || isAppMonitoringWithCSI || isReadOnlyHostMonitoring
}

func (dk *DynaKube) NeedAppInjection() bool {
	return dk.CloudNativeFullstackMode() || dk.ApplicationMonitoringMode()
}

func (dk *DynaKube) CustomOneAgentImage() string {
	switch {
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.Image
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.Image
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.Image
	}
	return ""
}

func (dk *DynaKube) CustomCodeModulesImage() string {
	if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.CodeModulesImage
	} else if dk.ApplicationMonitoringMode() && dk.NeedsCSIDriver() {
		return dk.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage
	}
	return ""
}

func (dk *DynaKube) InitResources() *corev1.ResourceRequirements {
	if dk.ApplicationMonitoringMode() {
		return &dk.Spec.OneAgent.ApplicationMonitoring.InitResources
	} else if dk.CloudNativeFullstackMode() {
		return &dk.Spec.OneAgent.CloudNativeFullStack.InitResources
	}
	return nil
}

func (dk *DynaKube) OneAgentResources() *corev1.ResourceRequirements {
	switch {
	case dk.ClassicFullStackMode():
		return &dk.Spec.OneAgent.ClassicFullStack.OneAgentResources
	case dk.HostMonitoringMode():
		return &dk.Spec.OneAgent.HostMonitoring.OneAgentResources
	case dk.CloudNativeFullstackMode():
		return &dk.Spec.OneAgent.CloudNativeFullStack.OneAgentResources
	}
	return nil
}

func (dk *DynaKube) NodeSelector() map[string]string {
	switch {
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.NodeSelector
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.NodeSelector
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.NodeSelector
	}
	return nil
}

func (dk *DynaKube) CustomOneAgentVersion() string {
	switch {
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.Version
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.Version
	case dk.ApplicationMonitoringMode():
		return dk.Spec.OneAgent.ApplicationMonitoring.Version
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.Version
	}
	return ""
}

// CodeModulesVersion does not take dynakube.Version into account when using cloudNative to avoid confusion
func (dk *DynaKube) CodeModulesVersion() string {
	if !dk.CloudNativeFullstackMode() && !dk.ApplicationMonitoringMode() {
		return ""
	}
	if dk.CustomCodeModulesImage() != "" {
		codeModulesImage := dk.CustomCodeModulesImage()
		return getRawImageTag(codeModulesImage)
	}
	if dk.CustomCodeModulesVersion() != "" {
		return dk.CustomCodeModulesVersion()
	}
	return dk.Status.CodeModules.Version
}

func (dk *DynaKube) CustomCodeModulesVersion() string {
	if !dk.ApplicationMonitoringMode() {
		return ""
	}
	return dk.CustomOneAgentVersion()
}

func (dk *DynaKube) NamespaceSelector() *metav1.LabelSelector {
	return &dk.Spec.NamespaceSelector
}

// OneAgentImage returns the immutable OneAgent image to be used with the DynaKube instance.
func (dk *DynaKube) OneAgentImage() string {
	oneAgentImage := dk.CustomOneAgentImage()
	if oneAgentImage != "" {
		return oneAgentImage
	}
	return dk.DefaultOneAgentImage()
}

func (dk *DynaKube) DefaultOneAgentImage() string {
	if dk.Spec.APIURL == "" {
		return ""
	}

	tag := api.LatestTag
	if version := dk.CustomOneAgentVersion(); version != "" {
		truncatedVersion := truncateBuildDate(version)
		tag = truncatedVersion
	}

	apiUrlHost := dk.ApiUrlHost()
	if apiUrlHost == "" {
		return ""
	}

	return fmt.Sprintf("%s/linux/oneagent:%s", apiUrlHost, tag)
}

func truncateBuildDate(version string) string {
	const versionSeparator = "."
	const buildDateIndex = 3

	if strings.Count(version, versionSeparator) >= buildDateIndex {
		splitVersion := strings.Split(version, versionSeparator)
		truncatedVersion := strings.Join(splitVersion[:buildDateIndex], versionSeparator)

		return truncatedVersion
	}

	return version
}

// Tokens returns the name of the Secret to be used for tokens.
func (dk *DynaKube) Tokens() string {
	if tkns := dk.Spec.Tokens; tkns != "" {
		return tkns
	}
	return dk.Name
}

// TenantUUIDFromApiUrl gets the tenantUUID from the ApiUrl present in the struct, if the tenant is aliased then the alias will be returned
func (dk *DynaKube) TenantUUIDFromApiUrl() (string, error) {
	return tenantUUID(dk.Spec.APIURL)
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

func (dk *DynaKube) HostGroup() string {
	var hostGroup string
	if dk.CloudNativeFullstackMode() && dk.Spec.OneAgent.CloudNativeFullStack.Args != nil {
		for _, arg := range dk.Spec.OneAgent.CloudNativeFullStack.Args {
			key, value := splitArg(arg)
			if key == "--set-host-group" {
				hostGroup = value
				break
			}
		}
	}
	return hostGroup
}

// UseActiveGateAuthToken returns if the activeGate should get an authToken mounted
func (dk *DynaKube) UseActiveGateAuthToken() bool {
	return dk.FeatureActiveGateAuthToken() && dk.NeedsActiveGate()
}

func splitArg(arg string) (key, value string) {
	split := strings.Split(arg, "=")
	if len(split) != 2 {
		return
	}
	key = split[0]
	value = split[1]
	return
}

func getRawImageTag(imageURI string) string {
	if !strings.Contains(imageURI, ":") {
		return api.LatestTag
	}
	splitURI := strings.Split(imageURI, ":")
	return splitURI[len(splitURI)-1]
}

func (dk *DynaKube) GetOneAgentEnvironment() []corev1.EnvVar {
	switch {
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.Env
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.Env
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.Env
	}
	return []corev1.EnvVar{}
}

func (dk *DynaKube) IsOneAgentConnectionInfoUpdateAllowed(timeProvider *timeprovider.Provider) bool {
	return timeProvider.IsOutdated(&dk.Status.DynatraceApi.LastOneAgentConnectionInfoRequest, dk.FeatureApiRequestThreshold())
}

func (dk *DynaKube) IsActiveGateConnectionInfoUpdateAllowed(timeProvider *timeprovider.Provider) bool {
	return timeProvider.IsOutdated(&dk.Status.DynatraceApi.LastActiveGateConnectionInfoRequest, dk.FeatureApiRequestThreshold())
}

func (dk *DynaKube) IsTokenScopeVerificationAllowed(timeProvider *timeprovider.Provider) bool {
	return timeProvider.IsOutdated(&dk.Status.DynatraceApi.LastTokenScopeRequest, dk.FeatureApiRequestThreshold())
}
