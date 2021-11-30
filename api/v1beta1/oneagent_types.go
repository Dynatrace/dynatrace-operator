package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

type OneAgentMode string

type OneAgentSpec struct {
	// Optional: enable classic fullstack monitoring and change its settings
	// Cannot be used in conjunction with cloud-native fullstack monitoring, application monitoring or host monitoring
	// +nullable
	ClassicFullStack *ClassicFullStackSpec `json:"classicFullStack,omitempty"`

	// Optional: (Preview feature) enable application-only monitoring and change its settings
	// Cannot be used in conjunction with cloud-native fullstack monitoring, classic fullstack monitoring or host monitoring
	// +nullable
	ApplicationMonitoring *ApplicationMonitoringSpec `json:"applicationMonitoring,omitempty"`

	// Optional: enable host monitoring and change its settings
	// Cannot be used in conjunction with cloud-native fullstack monitoring, classic fullstack monitoring or application monitoring
	// +nullable
	HostMonitoring *HostMonitoringSpec `json:"hostMonitoring,omitempty"`

	// Optional: (Preview feature) enable cloud-native fullstack monitoring and change its settings
	// Cannot be used in conjunction with classic fullstack monitoring, application monitoring or host monitoring
	// +nullable
	CloudNativeFullStack *CloudNativeFullStackSpec `json:"cloudNativeFullStack,omitempty"`
}

type CloudNativeFullStackSpec struct {

	// Optional: If specified, indicates the OneAgent version to use
	// Defaults to latest
	// Example: {major.minor.release} - 1.200.0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent version",order=11,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Version string `json:"version,omitempty"`

	HostInjectSpec   `json:",inline"`
	AppInjectionSpec `json:",inline"`
}

type ClassicFullStackSpec struct {
	// Optional: the Dynatrace installer container image
	// Defaults to docker.io/dynatrace/oneagent:latest for Kubernetes and to registry.connect.redhat.com/dynatrace/oneagent for OpenShift
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=12,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Optional: If specified, indicates the OneAgent version to use
	// Defaults to latest
	// Example: {major.minor.release} - 1.200.0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent version",order=11,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Version string `json:"version,omitempty"`

	HostInjectSpec `json:",inline"`
}

type HostMonitoringSpec struct {
	// Optional: the Dynatrace installer container image
	// Defaults to docker.io/dynatrace/oneagent:latest for Kubernetes and to registry.connect.redhat.com/dynatrace/oneagent for OpenShift
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=12,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Optional: If specified, indicates the OneAgent version to use
	// Defaults to latest
	// Example: {major.minor.release} - 1.200.0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent version",order=11,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Version string `json:"version,omitempty"`

	HostInjectSpec `json:",inline"`
}

type HostInjectSpec struct {

	// Node selector to control the selection of nodes (optional)
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=17,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Optional: If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that
	// name. If not specified the setting will be removed from the DaemonSet.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class name",order=23,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:PriorityClass"}
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Optional: set tolerations for the OneAgent pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations",order=18,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Optional: define resources requests and limits for single pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=20,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	OneAgentResources corev1.ResourceRequirements `json:"oneAgentResources,omitempty"`

	// Optional: Arguments to the OneAgent installer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent installer arguments",order=21,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +listType=set
	Args []string `json:"args,omitempty"`

	// Optional: List of environment variables to set for the installer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent environment variable installer arguments",order=22,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Optional: Enables automatic restarts of OneAgent pods in case a new version is available
	// Defaults to true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Automatically update Agent",order=13,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	AutoUpdate *bool `json:"autoUpdate,omitempty"`

	// Optional: Sets DNS Policy for the OneAgent pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DNS Policy",order=24,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// Optional: Adds additional labels for the OneAgent pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Labels",order=26,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Labels map[string]string `json:"labels,omitempty"`
}

type ApplicationMonitoringSpec struct {
	AppInjectionSpec `json:",inline"`

	// Optional: If specified, indicates the OneAgent version to use
	// Defaults to latest
	// Example: {major.minor.release} - 1.200.0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent version",order=11,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Version string `json:"version,omitempty"`

	// Optional: If you want to use CSIDriver; disable if your cluster does not have 'nodes' to fall back to the volume approach.
	UseCSIDriver *bool `json:"useCSIDriver,omitempty"`
}

type AppInjectionSpec struct {
	// Optional: define resources requests and limits for the initContainer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=15,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	InitResources corev1.ResourceRequirements `json:"initResources,omitempty"`
}
