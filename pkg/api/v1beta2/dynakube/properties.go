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

package dynakube

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// MaxNameLength is the maximum length of a DynaKube's name, we tend to add suffixes to the name to avoid name collisions for resources related to the DynaKube. (example: dkName-activegate-<some-hash>)
	// The limit is necessary because kubernetes uses the name of some resources (ActiveGate StatefulSet) for the label value, which has a limit of 63 characters. (see https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)
	MaxNameLength = 40

	// PullSecretSuffix is the suffix appended to the DynaKube name to n.
	PullSecretSuffix                        = "-pull-secret"
	ActiveGateTenantSecretSuffix            = "-activegate-tenant-secret"
	OneAgentTenantSecretSuffix              = "-oneagent-tenant-secret"
	OneAgentConnectionInfoConfigMapSuffix   = "-oneagent-connection-info"
	ActiveGateConnectionInfoConfigMapSuffix = "-activegate-connection-info"
	AuthTokenSecretSuffix                   = "-activegate-authtoken-secret"
	PodNameOsAgent                          = "oneagent"

	DefaultActiveGateImageRegistrySubPath = "/linux/activegate"
	DefaultOneAgentImageRegistrySubPath   = "/linux/oneagent"
)

// ApiUrl is a getter for dk.Spec.APIURL.
func (dk *DynaKube) ApiUrl() string {
	return dk.Spec.APIURL
}

func (dk *DynaKube) Conditions() *[]metav1.Condition { return &dk.Status.Conditions }

// ApiUrlHost returns the host of dk.Spec.APIURL
// E.g. if the APIURL is set to "https://my-tenant.dynatrace.com/api", it returns "my-tenant.dynatrace.com"
// If the URL cannot be parsed, it returns an empty string.
func (dk *DynaKube) ApiUrlHost() string {
	parsedUrl, err := url.Parse(dk.ApiUrl())
	if err != nil {
		return ""
	}

	return parsedUrl.Host
}

// NeedsActiveGate returns true when a feature requires ActiveGate instances.
func (dk *DynaKube) NeedsActiveGate() bool {
	return dk.ActiveGateMode()
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

// ClassicFullStackMode returns true when classic fullstack section is used.
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

func (dk *DynaKube) NeedsActiveGateServicePorts() bool {
	return dk.IsRoutingActiveGateEnabled() ||
		dk.IsApiActiveGateEnabled() ||
		dk.IsMetricsIngestActiveGateEnabled()
}

func (dk *DynaKube) NeedsActiveGateService() bool {
	return dk.NeedsActiveGateServicePorts()
}

func (dk *DynaKube) HasActiveGateCaCert() bool {
	return dk.ActiveGateMode() && dk.Spec.ActiveGate.TlsSecretName != ""
}

func (dk *DynaKube) NeedsOneAgentPrivileged() bool {
	return dk.FeatureOneAgentPrivileged()
}

func (dk *DynaKube) NeedsOneAgentProbe() bool {
	return dk.Status.OneAgent.Healthcheck != nil
}

// ShouldAutoUpdateOneAgent returns true if the Operator should update OneAgent instances automatically.
func (dk *DynaKube) ShouldAutoUpdateOneAgent() bool {
	switch {
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.AutoUpdate
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.AutoUpdate
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.AutoUpdate
	default:
		return false
	}
}

// ActivegateTenantSecret returns the name of the secret containing tenant UUID, token and communication endpoints for ActiveGate.
func (dk *DynaKube) ActivegateTenantSecret() string {
	return dk.Name + ActiveGateTenantSecretSuffix
}

// OneagentTenantSecret returns the name of the secret containing the token for the OneAgent.
func (dk *DynaKube) OneagentTenantSecret() string {
	return dk.Name + OneAgentTenantSecretSuffix
}

// ActiveGateAuthTokenSecret returns the name of the secret containing the ActiveGateAuthToken, which is mounted to the AGs.
func (dk *DynaKube) ActiveGateAuthTokenSecret() string {
	return dk.Name + AuthTokenSecretSuffix
}

func (dk *DynaKube) ActiveGateConnectionInfoConfigMapName() string {
	return dk.Name + ActiveGateConnectionInfoConfigMapSuffix
}

func (dk *DynaKube) OneAgentConnectionInfoConfigMapName() string {
	return dk.Name + OneAgentConnectionInfoConfigMapSuffix
}

// PullSecretName returns the name of the pull secret to be used for immutable images.
func (dk *DynaKube) PullSecretName() string {
	if dk.Spec.CustomPullSecret != "" {
		return dk.Spec.CustomPullSecret
	}

	return dk.Name + PullSecretSuffix
}

// PullSecretWithoutData returns a secret which can be used to query the actual secrets data from the cluster.
func (dk *DynaKube) PullSecretWithoutData() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.PullSecretName(),
			Namespace: dk.Namespace,
		},
	}
}

func (dk *DynaKube) NeedsReadOnlyOneAgents() bool {
	return dk.HostMonitoringMode() || dk.CloudNativeFullstackMode()
}

func (dk *DynaKube) NeedsCSIDriver() bool {
	isAppMonitoringWithCSI := dk.ApplicationMonitoringMode() &&
		dk.Spec.OneAgent.ApplicationMonitoring.UseCSIDriver

	return dk.CloudNativeFullstackMode() || isAppMonitoringWithCSI || dk.HostMonitoringMode()
}

func (dk *DynaKube) NeedAppInjection() bool {
	return dk.CloudNativeFullstackMode() || dk.ApplicationMonitoringMode()
}

func (dk *DynaKube) InitResources() *corev1.ResourceRequirements {
	if dk.ApplicationMonitoringMode() {
		return dk.Spec.OneAgent.ApplicationMonitoring.InitResources
	} else if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.InitResources
	}

	return nil
}

func (dk *DynaKube) OneAgentNamespaceSelector() *metav1.LabelSelector {
	switch {
	case dk.CloudNativeFullstackMode():
		return &dk.Spec.OneAgent.CloudNativeFullStack.NamespaceSelector
	case dk.ApplicationMonitoringMode():
		return &dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector
	}

	return nil
}

func (dk *DynaKube) MetadataEnrichmentNamespaceSelector() *metav1.LabelSelector {
	return &dk.Spec.MetadataEnrichment.NamespaceSelector
}

func (dk *DynaKube) OneAgentSecCompProfile() string {
	switch {
	case dk.CloudNativeFullstackMode():
		return dk.Spec.OneAgent.CloudNativeFullStack.SecCompProfile
	case dk.HostMonitoringMode():
		return dk.Spec.OneAgent.HostMonitoring.SecCompProfile
	case dk.ClassicFullStackMode():
		return dk.Spec.OneAgent.ClassicFullStack.SecCompProfile
	default:
		return ""
	}
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

// CodeModulesVersion provides version set in Status for the CodeModules.
func (dk *DynaKube) CodeModulesVersion() string {
	return dk.Status.CodeModules.Version
}

// CodeModulesImage provides the image reference set in Status for the CodeModules.
// Format: repo@sha256:digest.
func (dk *DynaKube) CodeModulesImage() string {
	return dk.Status.CodeModules.ImageID
}

// CustomCodeModulesImage provides the image reference for the CodeModules provided in the Spec.
func (dk *DynaKube) CustomCodeModulesImage() string {
	if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.CodeModulesImage
	} else if dk.ApplicationMonitoringMode() && dk.NeedsCSIDriver() {
		return dk.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage
	}

	return ""
}

// CustomCodeModulesVersion provides the version for the CodeModules provided in the Spec.
func (dk *DynaKube) CustomCodeModulesVersion() string {
	if !dk.ApplicationMonitoringMode() {
		return ""
	}

	return dk.CustomOneAgentVersion()
}

// OneAgentImage provides the image reference set in Status for the OneAgent.
// Format: repo@sha256:digest.
func (dk *DynaKube) OneAgentImage() string {
	return dk.Status.OneAgent.ImageID
}

// OneAgentVersion provides version set in Status for the OneAgent.
func (dk *DynaKube) OneAgentVersion() string {
	return dk.Status.OneAgent.Version
}

// CustomOneAgentVersion provides the version for the OneAgent provided in the Spec.
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

// CustomOneAgentImage provides the image reference for the OneAgent provided in the Spec.
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

// DefaultOneAgentImage provides the image reference for the OneAgent from tenant registry.
func (dk *DynaKube) DefaultOneAgentImage(version string) string {
	apiUrlHost := dk.ApiUrlHost()
	if apiUrlHost == "" {
		return ""
	}

	truncatedVersion := dtversion.ToImageTag(version)
	tag := truncatedVersion

	if !strings.Contains(tag, api.RawTag) {
		tag += "-" + api.RawTag
	}

	return apiUrlHost + DefaultOneAgentImageRegistrySubPath + ":" + tag
}

// Tokens returns the name of the Secret to be used for tokens.
func (dk *DynaKube) Tokens() string {
	if tkns := dk.Spec.Tokens; tkns != "" {
		return tkns
	}

	return dk.Name
}

// TenantUUIDFromApiUrl gets the tenantUUID from the ApiUrl present in the struct, if the tenant is aliased then the alias will be returned.
func (dk *DynaKube) TenantUUIDFromApiUrl() (string, error) {
	return tenantUUID(dk.Spec.APIURL)
}

func (dk *DynaKube) ApiRequestThreshold() time.Duration {
	if dk.Spec.DynatraceApiRequestThreshold < 0 {
		dk.Spec.DynatraceApiRequestThreshold = DefaultMinRequestThresholdMinutes
	}

	return time.Duration(dk.Spec.DynatraceApiRequestThreshold) * time.Minute
}

func (dk *DynaKube) MetadataEnrichmentEnabled() bool {
	return dk.Spec.MetadataEnrichment.Enabled
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
	if dk.Spec.OneAgent.HostGroup != "" {
		return dk.Spec.OneAgent.HostGroup
	}

	return dk.HostGroupAsParam()
}

func (dk *DynaKube) HostGroupAsParam() string {
	var hostGroup string

	var args []string

	switch {
	case dk.CloudNativeFullstackMode() && dk.Spec.OneAgent.CloudNativeFullStack.Args != nil:
		args = dk.Spec.OneAgent.CloudNativeFullStack.Args
	case dk.ClassicFullStackMode() && dk.Spec.OneAgent.ClassicFullStack.Args != nil:
		args = dk.Spec.OneAgent.ClassicFullStack.Args
	case dk.HostMonitoringMode() && dk.Spec.OneAgent.HostMonitoring.Args != nil:
		args = dk.Spec.OneAgent.HostMonitoring.Args
	}

	for _, arg := range args {
		key, value := splitArg(arg)
		if key == "--set-host-group" {
			hostGroup = value

			break
		}
	}

	return hostGroup
}

func splitArg(arg string) (key, value string) {
	split := strings.Split(arg, "=")

	const expectedLen = 2

	if len(split) != expectedLen {
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

// +kubebuilder:object:generate=false
type RequestAllowedChecker func(timeProvider *timeprovider.Provider) bool

func (dk *DynaKube) IsTokenScopeVerificationAllowed(timeProvider *timeprovider.Provider) bool {
	return timeProvider.IsOutdated(&dk.Status.DynatraceApi.LastTokenScopeRequest, dk.ApiRequestThreshold())
}

func (dk *DynaKube) IsOneAgentCommunicationRouteClear() bool {
	return len(dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts) > 0
}
