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

package v1

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/dtclient"
	corev1 "k8s.io/api/core/v1"
)

const (
	// PullSecretSuffix is the suffix appended to the DynaKube name to n.
	PullSecretSuffix = "-pull-secret"
)

// NeedsActiveGate returns true when a feature requires ActiveGate instances.
func (dk *DynaKube) NeedsActiveGate() bool {
	return dk.Spec.KubernetesMonitoringSpec.Enabled || dk.Spec.RoutingSpec.Enabled
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

// HostMonitoringMode returns true when host monitoring section is used.
func (dk *DynaKube) ClassicFullStackMode() bool {
	return dk.Spec.OneAgent != OneAgentSpec{} && dk.Spec.OneAgent.ClassicFullStack != nil
}

// NeedsOneAgent returns true when a feature requires OneAgent instances.
func (dk *DynaKube) NeedsOneAgent() bool {
	return dk.ClassicFullStackMode() || dk.CloudNativeFullstackMode()
}

// ShouldAutoUpdateOneAgent returns true if the Operator should update OneAgent instances automatically.
func (dk *DynaKube) ShouldAutoUpdateOneAgent() bool {
	if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.AutoUpdate == nil || *dk.Spec.OneAgent.CloudNativeFullStack.AutoUpdate
	} else if dk.ClassicFullStackMode() {
		return dk.Spec.OneAgent.ClassicFullStack.AutoUpdate == nil || *dk.Spec.OneAgent.ClassicFullStack.AutoUpdate
	}
	return false
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
	if dk.Spec.ActiveGate.Image != "" {
		return dk.Spec.ActiveGate.Image
	}

	if dk.Spec.APIURL == "" {
		return ""
	}

	registry := buildImageRegistry(dk.Spec.APIURL)
	return fmt.Sprintf("%s/linux/activegate:latest", registry)
}

func (dk *DynaKube) ServerlessMode() bool {
	if dk.CloudNativeFullstackMode() {
		return dk.Spec.OneAgent.CloudNativeFullStack.ServerlessMode
	} else if dk.ApplicationMonitoringMode() {
		return dk.Spec.OneAgent.ApplicationMonitoring.ServerlessMode
	}
	return false
}

func (dk *DynaKube) Image() string {
	if dk.ClassicFullStackMode() {
		return dk.Spec.OneAgent.ClassicFullStack.Image
	} else if dk.HostMonitoringMode() {
		return dk.Spec.OneAgent.HostMonitoring.Image
	} else if dk.ApplicationMonitoringMode() {
		return dk.Spec.OneAgent.ApplicationMonitoring.Image
	}

	return ""
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
	if ver := dk.Version(); ver != "" {
		tag = ver
	}

	registry := buildImageRegistry(dk.Spec.APIURL)
	return fmt.Sprintf("%s/linux/oneagent:%s", registry, tag)
}

func buildImageRegistry(apiURL string) string {
	registry := strings.TrimPrefix(apiURL, "https://")
	registry = strings.TrimPrefix(registry, "http://")
	registry = strings.TrimSuffix(registry, "/api")
	return registry
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

func (dk *DynaKube) GetInstallationVolume() corev1.VolumeSource {
	if dk.CloudNativeFullstackMode() {
		return *getInstallationVolume(dk.Spec.OneAgent.CloudNativeFullStack.InstallationVolume)
	} else if dk.HostMonitoringMode() {
		return *getInstallationVolume(dk.Spec.OneAgent.HostMonitoring.InstallationVolume)
	}
	return corev1.VolumeSource{}
}

func getInstallationVolume(vs *corev1.VolumeSource) *corev1.VolumeSource {
	if vs == nil {
		return &corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		}
	}
	return vs
}
