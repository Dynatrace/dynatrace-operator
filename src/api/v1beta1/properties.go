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

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PullSecretSuffix is the suffix appended to the DynaKube name to n.
	PullSecretSuffix   = "-pull-secret"
	TenantSecretSuffix = "-activegate-tenant-secret"

	PodNameOsAgent = "oneagent"
)

// NeedsActiveGate returns true when a feature requires ActiveGate instances.
func (dk *DynaKube) NeedsActiveGate() bool {
	return dk.DeprecatedActiveGateMode() || dk.ActiveGateMode()
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

func (dk *DynaKube) KubernetesMonitoringMode() bool {
	return dk.IsActiveGateMode(KubeMonCapability.DisplayName) || dk.Spec.KubernetesMonitoring.Enabled
}

func (dk *DynaKube) NeedsStatsd() bool {
	return dk.IsActiveGateMode(StatsdIngestCapability.DisplayName)
}

func (dk *DynaKube) HasActiveGateCaCert() bool {
	return dk.ActiveGateMode() && dk.Spec.ActiveGate.TlsSecretName != ""
}

func (dk *DynaKube) hasProxy() bool {
	return dk.Spec.Proxy != nil && (dk.Spec.Proxy.Value != "" || dk.Spec.Proxy.ValueFrom != "")
}

func (dk *DynaKube) NeedsActiveGateProxy() bool {
	return !dk.FeatureActiveGateIgnoreProxy() && dk.hasProxy()
}

func (dk *DynaKube) NeedsOneAgentProxy() bool {
	return !dk.FeatureOneAgentIgnoreProxy() && dk.hasProxy()
}

// ShouldAutoUpdateOneAgent returns true if the Operator should update OneAgent instances automatically.
func (dk *DynaKube) ShouldAutoUpdateOneAgent() bool {
	if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.AutoUpdate == nil || *dk.Spec.OneAgent.CloudNativeFullStack.AutoUpdate
	} else if dk.HostMonitoringMode() {
		return dk.Spec.OneAgent.HostMonitoring.AutoUpdate == nil || *dk.Spec.OneAgent.HostMonitoring.AutoUpdate
	} else if dk.ClassicFullStackMode() {
		return dk.Spec.OneAgent.ClassicFullStack.AutoUpdate == nil || *dk.Spec.OneAgent.ClassicFullStack.AutoUpdate
	}
	return false
}

// AGTenantSecret returns the name of the secret containing tenant UUID, token and communication endpoints for ActiveGate
func (dk *DynaKube) AGTenantSecret() string {
	return dk.Name + TenantSecretSuffix
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
	return resolveImagePath(newActiveGateImagePath(dk))
}

// EecImage returns the Extension Controller image to be used with the dk DynaKube instance.
func (dk *DynaKube) EecImage() string {
	return resolveImagePath(newEecImagePath(dk))
}

// StatsdImage returns the StatsD data source image to be used with the dk DynaKube instance.
func (dk *DynaKube) StatsdImage() string {
	return resolveImagePath(newStatsdImagePath(dk))
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
func (dk *DynaKube) Image() string {
	if dk.ClassicFullStackMode() {
		return dk.Spec.OneAgent.ClassicFullStack.Image
	} else if dk.HostMonitoringMode() {
		return dk.Spec.OneAgent.HostMonitoring.Image
	} else if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.Image
	}
	return ""
}

func (dk *DynaKube) CodeModulesImage() string {
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
	if dk.ClassicFullStackMode() {
		return &dk.Spec.OneAgent.ClassicFullStack.OneAgentResources
	} else if dk.HostMonitoringMode() {
		return &dk.Spec.OneAgent.HostMonitoring.OneAgentResources
	} else if dk.CloudNativeFullstackMode() {
		return &dk.Spec.OneAgent.CloudNativeFullStack.OneAgentResources
	}
	return nil
}

func (dk *DynaKube) NodeSelector() map[string]string {
	if dk.ClassicFullStackMode() {
		return dk.Spec.OneAgent.ClassicFullStack.NodeSelector
	} else if dk.HostMonitoringMode() {
		return dk.Spec.OneAgent.HostMonitoring.NodeSelector
	} else if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.NodeSelector
	}
	return nil
}

func (dk *DynaKube) Version() string {
	if dk.ClassicFullStackMode() {
		return dk.Spec.OneAgent.ClassicFullStack.Version
	} else if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.Version
	} else if dk.ApplicationMonitoringMode() {
		return dk.Spec.OneAgent.ApplicationMonitoring.Version
	} else if dk.HostMonitoringMode() {
		return dk.Spec.OneAgent.HostMonitoring.Version
	}
	return ""
}

// The dynakube.Version is not take into account when using cloudNative to avoid confusion
func (dynakube DynaKube) CodeModulesVersion() string {
	if !dynakube.CloudNativeFullstackMode() && !dynakube.ApplicationMonitoringMode() {
		return ""
	}
	if dynakube.CodeModulesImage() != "" {
		codeModulesImage := dynakube.CodeModulesImage()
		return strings.Split(codeModulesImage, ":")[1]
	}
	if dynakube.Version() != "" && !dynakube.CloudNativeFullstackMode() {
		return dynakube.Version()
	}
	return dynakube.Status.LatestAgentVersionUnixPaas
}

func (dk *DynaKube) NamespaceSelector() *metav1.LabelSelector {
	return &dk.Spec.NamespaceSelector
}

// ImmutableOneAgentImage returns the immutable OneAgent image to be used with the dk DynaKube instance.
func (dk *DynaKube) ImmutableOneAgentImage() string {
	oneAgentImage := dk.Image()
	if oneAgentImage != "" {
		return oneAgentImage
	}

	if dk.Spec.APIURL == "" {
		return ""
	}

	tag := "latest"
	if version := dk.Version(); version != "" {
		truncatedVersion := truncateBuildDate(version)
		tag = truncatedVersion
	}

	registry := buildImageRegistry(dk.Spec.APIURL)
	return fmt.Sprintf("%s/linux/oneagent:%s", registry, tag)
}

func truncateBuildDate(version string) string {
	const versionSeperator = "."
	const buildDateIndex = 3

	if strings.Count(version, versionSeperator) >= buildDateIndex {
		splitVersion := strings.Split(version, versionSeperator)
		truncatedVersion := strings.Join(splitVersion[:buildDateIndex], versionSeperator)

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

func (dk *DynaKube) CommunicationHostForClient() dtclient.CommunicationHost {
	return dtclient.CommunicationHost(dk.Status.CommunicationHostForClient)
}

func (dk *DynaKube) ConnectionInfo() dtclient.ConnectionInfo {
	return dtclient.ConnectionInfo{
		CommunicationHosts: dk.CommunicationHosts(),
		TenantUUID:         dk.Status.ConnectionInfo.TenantUUID,
	}
}

func (dk *DynaKube) CommunicationHosts() []dtclient.CommunicationHost {
	var communicationHosts []dtclient.CommunicationHost
	for _, communicationHost := range dk.Status.ConnectionInfo.CommunicationHosts {
		communicationHosts = append(communicationHosts, dtclient.CommunicationHost(communicationHost))
	}
	return communicationHosts
}

func (dk *DynaKube) TenantUUID() (string, error) {
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

	return "", fmt.Errorf("problem getting tenant id from API URL '%s'", apiUrl)
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

func splitArg(arg string) (key, value string) {
	split := strings.Split(arg, "=")
	if len(split) != 2 {
		return
	}
	key = split[0]
	value = split[1]
	return
}
