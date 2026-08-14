// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OneAgentTenantSecretSuffix            = "-oneagent-tenant-secret"
	OneAgentConnectionInfoConfigMapSuffix = "-oneagent-connection-info"
	PodNameOSAgent                        = "oneagent"
	DefaultOneAgentImageRegistrySubPath   = "/linux/oneagent"
	StorageVolumeDefaultHostPath          = "/var/opt/dynatrace"
)

func NewOneAgent(spec *Spec, status *Status, codeModulesStatus *CodeModulesStatus, name string) *OneAgent {
	return &OneAgent{
		Spec:              spec,
		Status:            status,
		CodeModulesStatus: codeModulesStatus,

		name: name,
	}
}

// IsApplicationMonitoringMode returns true when application only section is used.
func (oa *OneAgent) IsApplicationMonitoringMode() bool {
	return oa.ApplicationMonitoring != nil
}

// IsCloudNativeFullstackMode returns true when cloud native fullstack section is used.
func (oa *OneAgent) IsCloudNativeFullstackMode() bool {
	return oa.CloudNativeFullStack != nil
}

// IsHostMonitoringMode returns true when host monitoring section is used.
func (oa *OneAgent) IsHostMonitoringMode() bool {
	return oa.HostMonitoring != nil
}

// IsClassicFullStackMode returns true when classic fullstack section is used.
func (oa *OneAgent) IsClassicFullStackMode() bool {
	return oa.ClassicFullStack != nil
}

// IsDaemonsetRequired returns true when a feature requires OneAgent instances.
func (oa *OneAgent) IsDaemonsetRequired() bool {
	return oa.IsClassicFullStackMode() || oa.IsCloudNativeFullstackMode() || oa.IsHostMonitoringMode()
}

func (oa *OneAgent) GetDaemonsetName() string {
	return fmt.Sprintf("%s-%s", oa.name, PodNameOSAgent)
}

func (oa *OneAgent) GetNamespaceSelector() *metav1.LabelSelector {
	switch {
	case oa.IsCloudNativeFullstackMode():
		return &oa.CloudNativeFullStack.NamespaceSelector
	case oa.IsApplicationMonitoringMode():
		return &oa.ApplicationMonitoring.NamespaceSelector
	default:
		return nil
	}
}
