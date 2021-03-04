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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ActiveGateSpec) DeepCopyInto(out *ActiveGateSpec) {
	*out = *in
	if in.AutoUpdate != nil {
		in, out := &in.AutoUpdate, &out.AutoUpdate
		*out = new(bool)
		**out = **in
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
func (in *CapabilityProperties) DeepCopyInto(out *CapabilityProperties) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.CustomProperties != nil {
		in, out := &in.CustomProperties, &out.CustomProperties
		*out = new(DynaKubeValueSource)
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
func (in *CodeModulesSpec) DeepCopyInto(out *CodeModulesSpec) {
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
	in.Volume.DeepCopyInto(&out.Volume)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CodeModulesSpec.
func (in *CodeModulesSpec) DeepCopy() *CodeModulesSpec {
	if in == nil {
		return nil
	}
	out := new(CodeModulesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynaKube) DeepCopyInto(out *DynaKube) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynaKube.
func (in *DynaKube) DeepCopy() *DynaKube {
	if in == nil {
		return nil
	}
	out := new(DynaKube)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DynaKube) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynaKubeList) DeepCopyInto(out *DynaKubeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DynaKube, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynaKubeList.
func (in *DynaKubeList) DeepCopy() *DynaKubeList {
	if in == nil {
		return nil
	}
	out := new(DynaKubeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DynaKubeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynaKubeProxy) DeepCopyInto(out *DynaKubeProxy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynaKubeProxy.
func (in *DynaKubeProxy) DeepCopy() *DynaKubeProxy {
	if in == nil {
		return nil
	}
	out := new(DynaKubeProxy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynaKubeSpec) DeepCopyInto(out *DynaKubeSpec) {
	*out = *in
	if in.Proxy != nil {
		in, out := &in.Proxy, &out.Proxy
		*out = new(DynaKubeProxy)
		**out = **in
	}
	in.ActiveGate.DeepCopyInto(&out.ActiveGate)
	in.OneAgent.DeepCopyInto(&out.OneAgent)
	in.CodeModules.DeepCopyInto(&out.CodeModules)
	in.InfraMonitoring.DeepCopyInto(&out.InfraMonitoring)
	in.ClassicFullStack.DeepCopyInto(&out.ClassicFullStack)
	in.RoutingSpec.DeepCopyInto(&out.RoutingSpec)
	in.KubernetesMonitoringSpec.DeepCopyInto(&out.KubernetesMonitoringSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynaKubeSpec.
func (in *DynaKubeSpec) DeepCopy() *DynaKubeSpec {
	if in == nil {
		return nil
	}
	out := new(DynaKubeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynaKubeStatus) DeepCopyInto(out *DynaKubeStatus) {
	*out = *in
	in.UpdatedTimestamp.DeepCopyInto(&out.UpdatedTimestamp)
	if in.LastAPITokenProbeTimestamp != nil {
		in, out := &in.LastAPITokenProbeTimestamp, &out.LastAPITokenProbeTimestamp
		*out = (*in).DeepCopy()
	}
	if in.LastPaaSTokenProbeTimestamp != nil {
		in, out := &in.LastPaaSTokenProbeTimestamp, &out.LastPaaSTokenProbeTimestamp
		*out = (*in).DeepCopy()
	}
	if in.LastClusterVersionProbeTimestamp != nil {
		in, out := &in.LastClusterVersionProbeTimestamp, &out.LastClusterVersionProbeTimestamp
		*out = (*in).DeepCopy()
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.ActiveGate = in.ActiveGate
	in.OneAgent.DeepCopyInto(&out.OneAgent)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynaKubeStatus.
func (in *DynaKubeStatus) DeepCopy() *DynaKubeStatus {
	if in == nil {
		return nil
	}
	out := new(DynaKubeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynaKubeValueSource) DeepCopyInto(out *DynaKubeValueSource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynaKubeValueSource.
func (in *DynaKubeValueSource) DeepCopy() *DynaKubeValueSource {
	if in == nil {
		return nil
	}
	out := new(DynaKubeValueSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FullStackSpec) DeepCopyInto(out *FullStackSpec) {
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
	if in.WaitReadySeconds != nil {
		in, out := &in.WaitReadySeconds, &out.WaitReadySeconds
		*out = new(uint16)
		**out = **in
	}
	in.Resources.DeepCopyInto(&out.Resources)
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
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.UseUnprivilegedMode != nil {
		in, out := &in.UseUnprivilegedMode, &out.UseUnprivilegedMode
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FullStackSpec.
func (in *FullStackSpec) DeepCopy() *FullStackSpec {
	if in == nil {
		return nil
	}
	out := new(FullStackSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesMonitoringSpec) DeepCopyInto(out *KubernetesMonitoringSpec) {
	*out = *in
	in.CapabilityProperties.DeepCopyInto(&out.CapabilityProperties)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesMonitoringSpec.
func (in *KubernetesMonitoringSpec) DeepCopy() *KubernetesMonitoringSpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesMonitoringSpec)
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
func (in *OneAgentSpec) DeepCopyInto(out *OneAgentSpec) {
	*out = *in
	if in.AutoUpdate != nil {
		in, out := &in.AutoUpdate, &out.AutoUpdate
		*out = new(bool)
		**out = **in
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
	if in.Instances != nil {
		in, out := &in.Instances, &out.Instances
		*out = make(map[string]OneAgentInstance, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.LastUpdateProbeTimestamp != nil {
		in, out := &in.LastUpdateProbeTimestamp, &out.LastUpdateProbeTimestamp
		*out = (*in).DeepCopy()
	}
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
func (in *RoutingSpec) DeepCopyInto(out *RoutingSpec) {
	*out = *in
	in.CapabilityProperties.DeepCopyInto(&out.CapabilityProperties)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RoutingSpec.
func (in *RoutingSpec) DeepCopy() *RoutingSpec {
	if in == nil {
		return nil
	}
	out := new(RoutingSpec)
	in.DeepCopyInto(out)
	return out
}
