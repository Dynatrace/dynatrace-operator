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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppInjectionSpec) DeepCopyInto(out *AppInjectionSpec) {
	*out = *in
	in.NamespaceSelector.DeepCopyInto(&out.NamespaceSelector)
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
func (in *ClassicFullStackSpec) DeepCopyInto(out *ClassicFullStackSpec) {
	*out = *in
	in.HostInjectSpec.DeepCopyInto(&out.HostInjectSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClassicFullStackSpec.
func (in *ClassicFullStackSpec) DeepCopy() *ClassicFullStackSpec {
	if in == nil {
		return nil
	}
	out := new(ClassicFullStackSpec)
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
	out.ActiveGate = in.ActiveGate
	in.OneAgent.DeepCopyInto(&out.OneAgent)
	in.RoutingSpec.DeepCopyInto(&out.RoutingSpec)
	in.DataIngestSpec.DeepCopyInto(&out.DataIngestSpec)
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
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]corev1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AutoUpdate != nil {
		in, out := &in.AutoUpdate, &out.AutoUpdate
		*out = new(bool)
		**out = **in
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
func (in *HostMonitoringSpec) DeepCopyInto(out *HostMonitoringSpec) {
	*out = *in
	in.HostInjectSpec.DeepCopyInto(&out.HostInjectSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostMonitoringSpec.
func (in *HostMonitoringSpec) DeepCopy() *HostMonitoringSpec {
	if in == nil {
		return nil
	}
	out := new(HostMonitoringSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OneAgentSpec) DeepCopyInto(out *OneAgentSpec) {
	*out = *in
	if in.CloudNativeFullStack != nil {
		in, out := &in.CloudNativeFullStack, &out.CloudNativeFullStack
		*out = new(CloudNativeFullStackSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ClassicFullStack != nil {
		in, out := &in.ClassicFullStack, &out.ClassicFullStack
		*out = new(ClassicFullStackSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ApplicationMonitoring != nil {
		in, out := &in.ApplicationMonitoring, &out.ApplicationMonitoring
		*out = new(ApplicationMonitoringSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.HostMonitoring != nil {
		in, out := &in.HostMonitoring, &out.HostMonitoring
		*out = new(HostMonitoringSpec)
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
