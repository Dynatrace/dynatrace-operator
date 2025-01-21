//go:build !ignore_autogenerated

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

package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/telemetryservice"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynaKube) DeepCopyInto(out *DynaKube) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.Status.DeepCopyInto(&out.Status)
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
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
func (in *DynaKubeSpec) DeepCopyInto(out *DynaKubeSpec) {
	*out = *in
	in.MetadataEnrichment.DeepCopyInto(&out.MetadataEnrichment)
	if in.Proxy != nil {
		in, out := &in.Proxy, &out.Proxy
		*out = new(value.Source)
		**out = **in
	}
	if in.LogMonitoring != nil {
		in, out := &in.LogMonitoring, &out.LogMonitoring
		*out = new(logmonitoring.Spec)
		(*in).DeepCopyInto(*out)
	}
	if in.Kspm != nil {
		in, out := &in.Kspm, &out.Kspm
		*out = new(kspm.Spec)
		**out = **in
	}
	if in.DynatraceApiRequestThreshold != nil {
		in, out := &in.DynatraceApiRequestThreshold, &out.DynatraceApiRequestThreshold
		*out = new(uint16)
		**out = **in
	}
	if in.Extensions != nil {
		in, out := &in.Extensions, &out.Extensions
		*out = new(ExtensionsSpec)
		**out = **in
	}
	if in.TelemetryService != nil {
		in, out := &in.TelemetryService, &out.TelemetryService
		*out = new(telemetryservice.Spec)
		(*in).DeepCopyInto(*out)
	}
	in.OneAgent.DeepCopyInto(&out.OneAgent)
	in.MetadataEnrichment.DeepCopyInto(&out.MetadataEnrichment)
	in.Templates.DeepCopyInto(&out.Templates)
	in.ActiveGate.DeepCopyInto(&out.ActiveGate)
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
	in.OneAgent.DeepCopyInto(&out.OneAgent)
	in.ActiveGate.DeepCopyInto(&out.ActiveGate)
	in.CodeModules.DeepCopyInto(&out.CodeModules)
	in.MetadataEnrichment.DeepCopyInto(&out.MetadataEnrichment)
	out.Kspm = in.Kspm
	in.UpdatedTimestamp.DeepCopyInto(&out.UpdatedTimestamp)
	in.DynatraceApi.DeepCopyInto(&out.DynatraceApi)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
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
func (in *EnrichmentRule) DeepCopyInto(out *EnrichmentRule) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnrichmentRule.
func (in *EnrichmentRule) DeepCopy() *EnrichmentRule {
	if in == nil {
		return nil
	}
	out := new(EnrichmentRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtensionExecutionControllerSpec) DeepCopyInto(out *ExtensionExecutionControllerSpec) {
	*out = *in
	if in.PersistentVolumeClaim != nil {
		in, out := &in.PersistentVolumeClaim, &out.PersistentVolumeClaim
		*out = new(corev1.PersistentVolumeClaimSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	out.ImageRef = in.ImageRef
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]corev1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtensionExecutionControllerSpec.
func (in *ExtensionExecutionControllerSpec) DeepCopy() *ExtensionExecutionControllerSpec {
	if in == nil {
		return nil
	}
	out := new(ExtensionExecutionControllerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtensionsSpec) DeepCopyInto(out *ExtensionsSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtensionsSpec.
func (in *ExtensionsSpec) DeepCopy() *ExtensionsSpec {
	if in == nil {
		return nil
	}
	out := new(ExtensionsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetadataEnrichment) DeepCopyInto(out *MetadataEnrichment) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	in.NamespaceSelector.DeepCopyInto(&out.NamespaceSelector)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetadataEnrichment.
func (in *MetadataEnrichment) DeepCopy() *MetadataEnrichment {
	if in == nil {
		return nil
	}
	out := new(MetadataEnrichment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetadataEnrichmentStatus) DeepCopyInto(out *MetadataEnrichmentStatus) {
	*out = *in
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]EnrichmentRule, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetadataEnrichmentStatus.
func (in *MetadataEnrichmentStatus) DeepCopy() *MetadataEnrichmentStatus {
	if in == nil {
		return nil
	}
	out := new(MetadataEnrichmentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpenTelemetryCollectorSpec) DeepCopyInto(out *OpenTelemetryCollectorSpec) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	out.ImageRef = in.ImageRef
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]corev1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpenTelemetryCollectorSpec.
func (in *OpenTelemetryCollectorSpec) DeepCopy() *OpenTelemetryCollectorSpec {
	if in == nil {
		return nil
	}
	out := new(OpenTelemetryCollectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TemplatesSpec) DeepCopyInto(out *TemplatesSpec) {
	*out = *in
	if in.LogMonitoring != nil {
		in, out := &in.LogMonitoring, &out.LogMonitoring
		*out = new(logmonitoring.TemplateSpec)
		(*in).DeepCopyInto(*out)
	}
	in.KspmNodeConfigurationCollector.DeepCopyInto(&out.KspmNodeConfigurationCollector)
	in.OpenTelemetryCollector.DeepCopyInto(&out.OpenTelemetryCollector)
	in.ExtensionExecutionController.DeepCopyInto(&out.ExtensionExecutionController)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TemplatesSpec.
func (in *TemplatesSpec) DeepCopy() *TemplatesSpec {
	if in == nil {
		return nil
	}
	out := new(TemplatesSpec)
	in.DeepCopyInto(out)
	return out
}
