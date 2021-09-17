package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OneAgentMode string

type OneAgentSpec struct {
	// Optional: enable cloud-native fullstack monitoring and change its settings
	// Cannot be used in conjunction with classic fullstack monitoring or application-only monitoring
	// +nullable
	CloudNativeSpec *CloudNativeSpec `json:"cloudNativeFullStack,omitempty"`

	// Optional: enable classic fullstack monitoring and change its settings
	// Cannot be used in conjunction with cloud-native fullstack monitoring or application-only monitoring
	// +nullable
	ClassicSpec *ClassicFullStackSpec `json:"classicFullStack,omitempty"`

	// Optional: enable application-only monitoring and change its settings
	// Cannot be used in conjunction with cloud-native fullstack monitoring or classic fullstack monitoring
	// +nullable
	ApplicationOnlySpec *ApplicationOnlySpec `json:"applicationOnly,omitempty"`
}

type CloudNativeSpec struct {
	// Optional: the Dynatrace installer container image
	// Defaults to docker.io/dynatrace/oneagent:latest for Kubernetes and to registry.connect.redhat.com/dynatrace/oneagent for OpenShift
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=12,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Optional: set a node selector to limit on which nodes the pods are deployed
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=17,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector v1.NodeSelector `json:"nodeSelector,omitempty"`

	// Optional: set tolerations for the OneAgent pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations",order=18,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`

	// Optional: define resources requests and limits for single pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=20,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: Arguments to the OneAgent installer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent installer arguments",order=21,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +listType=set
	Args []string `json:"args,omitempty"`

	// Optional: List of environment variables to set for the installer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent environment variable installer arguments",order=22,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Env []v1.EnvVar `json:"env,omitempty"`

	// Optional: set a namespace selector to limit which namespaces are monitored
	// By default, all namespaces will be monitored
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace Selector",order=17,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Namespace"
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

type ClassicFullStackSpec struct {
	// Optional: the Dynatrace installer container image
	// Defaults to docker.io/dynatrace/oneagent:latest for Kubernetes and to registry.connect.redhat.com/dynatrace/oneagent for OpenShift
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=12,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Optional: set a node selector to limit on which nodes the pods are deployed
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=17,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector v1.NodeSelector `json:"nodeSelector,omitempty"`

	// Optional: set tolerations for the OneAgent pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations",order=18,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`

	// Optional: define resources requests and limits for single pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=20,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: Arguments to the OneAgent installer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent installer arguments",order=21,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +listType=set
	Args []string `json:"args,omitempty"`

	// Optional: List of environment variables to set for the installer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent environment variable installer arguments",order=22,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Env []v1.EnvVar `json:"env,omitempty"`
}

type ApplicationOnlySpec struct {
	// Optional: set a namespace selector to limit which namespaces are monitored
	// By default, all namespaces will be monitored
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace Selector",order=17,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Namespace"
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}
