//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActiveGate) DeepCopyInto(out *ActiveGate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActiveGate.
func (in *ActiveGate) DeepCopy() *ActiveGate {
	if in == nil {
		return nil
	}
	out := new(ActiveGate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActiveGate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActiveGateCapability) DeepCopyInto(out *ActiveGateCapability) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActiveGateCapability.
func (in *ActiveGateCapability) DeepCopy() *ActiveGateCapability {
	if in == nil {
		return nil
	}
	out := new(ActiveGateCapability)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActiveGateConnectionInfoStatus) DeepCopyInto(out *ActiveGateConnectionInfoStatus) {
	*out = *in
	in.ConnectionInfoStatus.DeepCopyInto(&out.ConnectionInfoStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActiveGateConnectionInfoStatus.
func (in *ActiveGateConnectionInfoStatus) DeepCopy() *ActiveGateConnectionInfoStatus {
	if in == nil {
		return nil
	}
	out := new(ActiveGateConnectionInfoStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActiveGateList) DeepCopyInto(out *ActiveGateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ActiveGate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActiveGateList.
func (in *ActiveGateList) DeepCopy() *ActiveGateList {
	if in == nil {
		return nil
	}
	out := new(ActiveGateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ActiveGateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActiveGateSpec) DeepCopyInto(out *ActiveGateSpec) {
	*out = *in
	in.Environment.DeepCopyInto(&out.Environment)
	if in.Capabilities != nil {
		in, out := &in.Capabilities, &out.Capabilities
		*out = make([]CapabilityDisplayName, len(*in))
		copy(*out, *in)
	}
	in.CapabilityProperties.DeepCopyInto(&out.CapabilityProperties)
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActiveGateSpec.
func (in *ActiveGateSpec) DeepCopy() *ActiveGateSpec {
	if in == nil {
		return nil
	}
	out := new(ActiveGateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActiveGateStatus) DeepCopyInto(out *ActiveGateStatus) {
	*out = *in
	in.VersionStatus.DeepCopyInto(&out.VersionStatus)
	in.ConnectionInfoStatus.DeepCopyInto(&out.ConnectionInfoStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActiveGateStatus.
func (in *ActiveGateStatus) DeepCopy() *ActiveGateStatus {
	if in == nil {
		return nil
	}
	out := new(ActiveGateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppInjectionSpec) DeepCopyInto(out *AppInjectionSpec) {
	*out = *in
	if in.InitResources != nil {
		in, out := &in.InitResources, &out.InitResources
		*out = new(v1.ResourceRequirements)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppInjectionSpec.
func (in *AppInjectionSpec) DeepCopy() *AppInjectionSpec {
	if in == nil {
		return nil
	}
	out := new(AppInjectionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationMonitoringSpec) DeepCopyInto(out *ApplicationMonitoringSpec) {
	*out = *in
	in.AppInjectionSpec.DeepCopyInto(&out.AppInjectionSpec)
	if in.UseCSIDriver != nil {
		in, out := &in.UseCSIDriver, &out.UseCSIDriver
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationMonitoringSpec.
func (in *ApplicationMonitoringSpec) DeepCopy() *ApplicationMonitoringSpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationMonitoringSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapabilityProperties) DeepCopyInto(out *CapabilityProperties) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.CustomProperties != nil {
		in, out := &in.CustomProperties, &out.CustomProperties
		*out = new(StringValueSource)
		**out = **in
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]v1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapabilityProperties.
func (in *CapabilityProperties) DeepCopy() *CapabilityProperties {
	if in == nil {
		return nil
	}
	out := new(CapabilityProperties)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudNativeFullStackSpec) DeepCopyInto(out *CloudNativeFullStackSpec) {
	*out = *in
	in.HostInjectSpec.DeepCopyInto(&out.HostInjectSpec)
	in.AppInjectionSpec.DeepCopyInto(&out.AppInjectionSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudNativeFullStackSpec.
func (in *CloudNativeFullStackSpec) DeepCopy() *CloudNativeFullStackSpec {
	if in == nil {
		return nil
	}
	out := new(CloudNativeFullStackSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CodeModulesStatus) DeepCopyInto(out *CodeModulesStatus) {
	*out = *in
	in.VersionStatus.DeepCopyInto(&out.VersionStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CodeModulesStatus.
func (in *CodeModulesStatus) DeepCopy() *CodeModulesStatus {
	if in == nil {
		return nil
	}
	out := new(CodeModulesStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CommunicationHostStatus) DeepCopyInto(out *CommunicationHostStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CommunicationHostStatus.
func (in *CommunicationHostStatus) DeepCopy() *CommunicationHostStatus {
	if in == nil {
		return nil
	}
	out := new(CommunicationHostStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConnectionInfoStatus) DeepCopyInto(out *ConnectionInfoStatus) {
	*out = *in
	in.LastRequest.DeepCopyInto(&out.LastRequest)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConnectionInfoStatus.
func (in *ConnectionInfoStatus) DeepCopy() *ConnectionInfoStatus {
	if in == nil {
		return nil
	}
	out := new(ConnectionInfoStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynatraceApiStatus) DeepCopyInto(out *DynatraceApiStatus) {
	*out = *in
	in.LastTokenScopeRequest.DeepCopyInto(&out.LastTokenScopeRequest)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynatraceApiStatus.
func (in *DynatraceApiStatus) DeepCopy() *DynatraceApiStatus {
	if in == nil {
		return nil
	}
	out := new(DynatraceApiStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Environment) DeepCopyInto(out *Environment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Environment.
func (in *Environment) DeepCopy() *Environment {
	if in == nil {
		return nil
	}
	out := new(Environment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Environment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvironmentList) DeepCopyInto(out *EnvironmentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Environment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvironmentList.
func (in *EnvironmentList) DeepCopy() *EnvironmentList {
	if in == nil {
		return nil
	}
	out := new(EnvironmentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EnvironmentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvironmentSpec) DeepCopyInto(out *EnvironmentSpec) {
	*out = *in
	if in.Proxy != nil {
		in, out := &in.Proxy, &out.Proxy
		*out = new(StringValueSource)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvironmentSpec.
func (in *EnvironmentSpec) DeepCopy() *EnvironmentSpec {
	if in == nil {
		return nil
	}
	out := new(EnvironmentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvironmentStatus) DeepCopyInto(out *EnvironmentStatus) {
	*out = *in
	in.UpdatedTimestamp.DeepCopyInto(&out.UpdatedTimestamp)
	in.ApiStatus.DeepCopyInto(&out.ApiStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvironmentStatus.
func (in *EnvironmentStatus) DeepCopy() *EnvironmentStatus {
	if in == nil {
		return nil
	}
	out := new(EnvironmentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvironmentValueSource) DeepCopyInto(out *EnvironmentValueSource) {
	*out = *in
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		*out = new(Environment)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvironmentValueSource.
func (in *EnvironmentValueSource) DeepCopy() *EnvironmentValueSource {
	if in == nil {
		return nil
	}
	out := new(EnvironmentValueSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostInjectSpec) DeepCopyInto(out *HostInjectSpec) {
	*out = *in
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.OneAgentResources.DeepCopyInto(&out.OneAgentResources)
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AutoUpdate != nil {
		in, out := &in.AutoUpdate, &out.AutoUpdate
		*out = new(bool)
		**out = **in
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostInjectSpec.
func (in *HostInjectSpec) DeepCopy() *HostInjectSpec {
	if in == nil {
		return nil
	}
	out := new(HostInjectSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgent) DeepCopyInto(out *OneAgent) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgent.
func (in *OneAgent) DeepCopy() *OneAgent {
	if in == nil {
		return nil
	}
	out := new(OneAgent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OneAgent) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentConnectionInfoStatus) DeepCopyInto(out *OneAgentConnectionInfoStatus) {
	*out = *in
	in.ConnectionInfoStatus.DeepCopyInto(&out.ConnectionInfoStatus)
	if in.CommunicationHosts != nil {
		in, out := &in.CommunicationHosts, &out.CommunicationHosts
		*out = make([]CommunicationHostStatus, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentConnectionInfoStatus.
func (in *OneAgentConnectionInfoStatus) DeepCopy() *OneAgentConnectionInfoStatus {
	if in == nil {
		return nil
	}
	out := new(OneAgentConnectionInfoStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentInstance) DeepCopyInto(out *OneAgentInstance) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentInstance.
func (in *OneAgentInstance) DeepCopy() *OneAgentInstance {
	if in == nil {
		return nil
	}
	out := new(OneAgentInstance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentList) DeepCopyInto(out *OneAgentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OneAgent, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentList.
func (in *OneAgentList) DeepCopy() *OneAgentList {
	if in == nil {
		return nil
	}
	out := new(OneAgentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OneAgentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentSpec) DeepCopyInto(out *OneAgentSpec) {
	*out = *in
	in.Environment.DeepCopyInto(&out.Environment)
	if in.ClassicFullStack != nil {
		in, out := &in.ClassicFullStack, &out.ClassicFullStack
		*out = new(HostInjectSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ApplicationMonitoring != nil {
		in, out := &in.ApplicationMonitoring, &out.ApplicationMonitoring
		*out = new(ApplicationMonitoringSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.HostMonitoring != nil {
		in, out := &in.HostMonitoring, &out.HostMonitoring
		*out = new(HostInjectSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.CloudNativeFullStack != nil {
		in, out := &in.CloudNativeFullStack, &out.CloudNativeFullStack
		*out = new(CloudNativeFullStackSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentSpec.
func (in *OneAgentSpec) DeepCopy() *OneAgentSpec {
	if in == nil {
		return nil
	}
	out := new(OneAgentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentStatus) DeepCopyInto(out *OneAgentStatus) {
	*out = *in
	in.VersionStatus.DeepCopyInto(&out.VersionStatus)
	if in.Instances != nil {
		in, out := &in.Instances, &out.Instances
		*out = make(map[string]OneAgentInstance, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.LastInstanceStatusUpdate != nil {
		in, out := &in.LastInstanceStatusUpdate, &out.LastInstanceStatusUpdate
		*out = (*in).DeepCopy()
	}
	in.ConnectionInfoStatus.DeepCopyInto(&out.ConnectionInfoStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OneAgentStatus.
func (in *OneAgentStatus) DeepCopy() *OneAgentStatus {
	if in == nil {
		return nil
	}
	out := new(OneAgentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StringValueSource) DeepCopyInto(out *StringValueSource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StringValueSource.
func (in *StringValueSource) DeepCopy() *StringValueSource {
	if in == nil {
		return nil
	}
	out := new(StringValueSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VersionStatus) DeepCopyInto(out *VersionStatus) {
	*out = *in
	if in.LastProbeTimestamp != nil {
		in, out := &in.LastProbeTimestamp, &out.LastProbeTimestamp
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VersionStatus.
func (in *VersionStatus) DeepCopy() *VersionStatus {
	if in == nil {
		return nil
	}
	out := new(VersionStatus)
	in.DeepCopyInto(out)
	return out
}
