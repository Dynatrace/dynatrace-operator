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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PullSecretSuffix is the suffix appended to the DynaKube name to n.
	PodNameOsAgent = "oneagent"
)

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

func (dk *DynaKube) NeedAppInjection() bool {
	return dk.CloudNativeFullstackMode() || dk.ApplicationMonitoringMode()
}

func (dk *DynaKube) NamespaceSelector() *metav1.LabelSelector {
	return &dk.Spec.NamespaceSelector
}
