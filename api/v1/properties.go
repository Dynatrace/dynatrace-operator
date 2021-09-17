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
)

const (
	// PullSecretSuffix is the suffix appended to the DynaKube name to n.
	PullSecretSuffix = "-pull-secret"
)

// NeedsActiveGate returns true when a feature requires ActiveGate instances.
func (dk *DynaKube) NeedsActiveGate() bool {
	return dk.Spec.KubernetesMonitoringSpec.Enabled || dk.Spec.RoutingSpec.Enabled
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

//
//// ImmutableOneAgentImage returns the immutable OneAgent image to be used with the dk DynaKube instance.
//func (dk *DynaKube) ImmutableOneAgentImage() string {
//	if dk.Spec.OneAgent.Image != "" {
//		return dk.Spec.OneAgent.Image
//	}
//
//	if dk.Spec.APIURL == "" {
//		return ""
//	}
//
//	tag := "latest"
//	if ver := dk.Spec.OneAgent.Version; ver != "" {
//		tag = ver
//	}
//
//	registry := buildImageRegistry(dk.Spec.APIURL)
//	return fmt.Sprintf("%s/linux/oneagent:%s", registry, tag)
//}

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

//
//func (readOnlySpec *ReadOnlySpec) GetInstallationVolume() v1.VolumeSource {
//	if readOnlySpec.InstallationVolume == nil {
//		return v1.VolumeSource{
//			EmptyDir: &v1.EmptyDirVolumeSource{},
//		}
//	}
//	return *readOnlySpec.InstallationVolume
//}
