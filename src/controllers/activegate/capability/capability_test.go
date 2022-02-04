package capability

import (
	"reflect"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	v1 "k8s.io/api/core/v1"
)

func Test_capabilityBase_Properties(t *testing.T) {
	props := &dynatracev1beta1.CapabilityProperties{}

	type fields struct {
		properties *dynatracev1beta1.CapabilityProperties
	}
	tests := []struct {
		name   string
		fields fields
		want   *dynatracev1beta1.CapabilityProperties
	}{
		{
			name: "properties address is preserved",
			fields: fields{
				properties: props,
			},
			want: &dynatracev1beta1.CapabilityProperties{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &capabilityBase{
				properties: tt.fields.properties,
			}
			if got := c.Properties(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("capabilityBase.Properties() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_capabilityBase_Configuration(t *testing.T) {
	conf := Configuration{
		SetDnsEntryPoint:     false,
		SetReadinessPort:     false,
		SetCommunicationPort: true,
		CreateService:        true,
		ServiceAccountOwner:  "accowner",
	}

	type fields struct {
		Configuration Configuration
	}
	tests := []struct {
		name   string
		fields fields
		want   Configuration
	}{
		{
			name: "configuration is correct",
			fields: fields{
				Configuration: conf,
			},
			want: conf,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &capabilityBase{
				Configuration: tt.fields.Configuration,
			}
			if got := c.Config(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("capabilityBase.Config() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_capabilityBase_GetModuleName(t *testing.T) {
	const expectedModuleName = "some_module"

	type fields struct {
		moduleName string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "module name is correct",
			fields: fields{
				moduleName: expectedModuleName,
			},
			want: expectedModuleName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &capabilityBase{
				shortName: tt.fields.moduleName,
			}
			if got := c.ShortName(); got != tt.want {
				t.Errorf("capabilityBase.ShortName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_capabilityBase_GetCapabilityName(t *testing.T) {
	const expectedCapabilityName = "capability_name"

	type fields struct {
		capabilityName string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "capability name is correct",
			fields: fields{
				capabilityName: expectedCapabilityName,
			},
			want: expectedCapabilityName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &capabilityBase{
				argName: tt.fields.capabilityName,
			}
			if got := c.ArgName(); got != tt.want {
				t.Errorf("capabilityBase.ArgName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateStatefulSetName(t *testing.T) {
	c := NewKubeMonCapability(nil)
	const instanceName = "testinstance"

	type args struct {
		capability   Capability
		instanceName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "",
			args: args{
				capability:   c,
				instanceName: instanceName,
			},
			want: instanceName + "-" + c.ShortName(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateStatefulSetName(tt.args.capability, tt.args.instanceName); got != tt.want {
				t.Errorf("CalculateStatefulSetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewKubeMonCapability(t *testing.T) {
	props := &dynatracev1beta1.CapabilityProperties{}
	dk := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
				CapabilityProperties: *props,
			},
		},
	}

	type args struct {
		dynakube *dynatracev1beta1.DynaKube
	}
	tests := []struct {
		name string
		args args
		want *KubeMonCapability
	}{
		{
			name: "default",
			args: args{
				dynakube: dk,
			},
			want: &KubeMonCapability{
				capabilityBase: capabilityBase{
					shortName:  dynatracev1beta1.KubeMonCapability.ShortName,
					argName:    dynatracev1beta1.KubeMonCapability.ArgumentName,
					properties: props,
					Configuration: Configuration{
						ServiceAccountOwner: "kubernetes-monitoring",
					},
					initContainersTemplates: []v1.Container{
						{
							Name:            initContainerTemplateName,
							ImagePullPolicy: v1.PullAlways,
							WorkingDir:      k8scrt2jksWorkingDir,
							Command:         []string{"/bin/bash"},
							Args:            []string{"-c", k8scrt2jksPath},
							VolumeMounts: []v1.VolumeMount{
								{
									ReadOnly:  false,
									Name:      trustStoreVolume,
									MountPath: activeGateSslPath,
								},
							},
						},
					},
					containerVolumeMounts: []v1.VolumeMount{{
						ReadOnly:  true,
						Name:      trustStoreVolume,
						MountPath: activeGateCacertsPath,
						SubPath:   k8sCertificateFile,
					}},
					volumes: []v1.Volume{{
						Name: trustStoreVolume,
						VolumeSource: v1.VolumeSource{
							EmptyDir: &v1.EmptyDirVolumeSource{},
						},
					}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewKubeMonCapability(tt.args.dynakube); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewKubeMonCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRoutingCapability(t *testing.T) {

	props := &dynatracev1beta1.CapabilityProperties{}
	dk := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			Routing: dynatracev1beta1.RoutingSpec{
				CapabilityProperties: *props,
			},
		},
	}

	type args struct {
		dynakube *dynatracev1beta1.DynaKube
	}
	tests := []struct {
		name string
		args args
		want *RoutingCapability
	}{
		{
			name: "default",
			args: args{
				dynakube: dk,
			},
			want: &RoutingCapability{
				capabilityBase: capabilityBase{
					shortName:  dynatracev1beta1.RoutingCapability.ShortName,
					argName:    dynatracev1beta1.RoutingCapability.ArgumentName,
					properties: props,
					Configuration: Configuration{
						SetDnsEntryPoint:     true,
						SetReadinessPort:     true,
						SetCommunicationPort: true,
						CreateService:        true,
						ServiceAccountOwner:  "",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRoutingCapability(tt.args.dynakube); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRoutingCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewMultiCapability(t *testing.T) {

	props := &dynatracev1beta1.CapabilityProperties{}

	type args struct {
		dynakube *dynatracev1beta1.DynaKube
	}
	tests := []struct {
		name string
		args args
		want *MultiCapability
	}{
		{
			name: "empty",
			args: args{
				dynakube: &dynatracev1beta1.DynaKube{
					Spec: dynatracev1beta1.DynaKubeSpec{},
				},
			},
			want: &MultiCapability{
				capabilityBase: capabilityBase{
					enabled:   false,
					shortName: MultiActiveGateName,
					Configuration: Configuration{
						CreateService: true,
					},
				},
			},
		},
		{
			name: "just routing",
			args: args{
				dynakube: &dynatracev1beta1.DynaKube{
					Spec: dynatracev1beta1.DynaKubeSpec{
						ActiveGate: dynatracev1beta1.ActiveGateSpec{
							Capabilities: []dynatracev1beta1.CapabilityDisplayName{
								dynatracev1beta1.RoutingCapability.DisplayName,
							},
						},
					},
				},
			},
			want: &MultiCapability{
				capabilityBase: capabilityBase{
					enabled:    true,
					shortName:  MultiActiveGateName,
					argName:    dynatracev1beta1.RoutingCapability.ArgumentName,
					properties: props,
					Configuration: Configuration{
						SetDnsEntryPoint:     true,
						SetReadinessPort:     true,
						SetCommunicationPort: true,
						CreateService:        true,
						ServiceAccountOwner:  "",
					},
				},
			},
		},
		{
			name: "just metrics-ingest",
			args: args{
				dynakube: &dynatracev1beta1.DynaKube{
					Spec: dynatracev1beta1.DynaKubeSpec{
						ActiveGate: dynatracev1beta1.ActiveGateSpec{
							Capabilities: []dynatracev1beta1.CapabilityDisplayName{
								dynatracev1beta1.MetricsIngestCapability.DisplayName,
							},
						},
					},
				},
			},
			want: &MultiCapability{
				capabilityBase: capabilityBase{
					enabled:    true,
					shortName:  MultiActiveGateName,
					argName:    dynatracev1beta1.MetricsIngestCapability.ArgumentName,
					properties: props,
					Configuration: Configuration{
						SetDnsEntryPoint:     true,
						SetReadinessPort:     true,
						SetCommunicationPort: true,
						CreateService:        true,
						ServiceAccountOwner:  "",
					},
				},
			},
		},
		{
			name: "just dynatrace-api",
			args: args{
				dynakube: &dynatracev1beta1.DynaKube{
					Spec: dynatracev1beta1.DynaKubeSpec{
						ActiveGate: dynatracev1beta1.ActiveGateSpec{
							Capabilities: []dynatracev1beta1.CapabilityDisplayName{
								dynatracev1beta1.DynatraceApiCapability.DisplayName,
							},
						},
					},
				},
			},
			want: &MultiCapability{
				capabilityBase: capabilityBase{
					enabled:    true,
					shortName:  MultiActiveGateName,
					argName:    dynatracev1beta1.DynatraceApiCapability.ArgumentName,
					properties: props,
					Configuration: Configuration{
						SetDnsEntryPoint:     true,
						SetReadinessPort:     true,
						SetCommunicationPort: true,
						CreateService:        true,
						ServiceAccountOwner:  "",
					},
				},
			},
		},
		{
			name: "just kubemon",
			args: args{
				dynakube: &dynatracev1beta1.DynaKube{
					Spec: dynatracev1beta1.DynaKubeSpec{
						ActiveGate: dynatracev1beta1.ActiveGateSpec{
							Capabilities: []dynatracev1beta1.CapabilityDisplayName{
								dynatracev1beta1.KubeMonCapability.DisplayName,
							},
						},
					},
				},
			},
			want: &MultiCapability{
				capabilityBase: capabilityBase{
					enabled:    true,
					shortName:  MultiActiveGateName,
					argName:    dynatracev1beta1.KubeMonCapability.ArgumentName,
					properties: props,
					Configuration: Configuration{
						ServiceAccountOwner: "kubernetes-monitoring",
					},
					initContainersTemplates: []v1.Container{
						{
							Name:            initContainerTemplateName,
							ImagePullPolicy: v1.PullAlways,
							WorkingDir:      k8scrt2jksWorkingDir,
							Command:         []string{"/bin/bash"},
							Args:            []string{"-c", k8scrt2jksPath},
							VolumeMounts: []v1.VolumeMount{
								{
									ReadOnly:  false,
									Name:      trustStoreVolume,
									MountPath: activeGateSslPath,
								},
							},
						},
					},
					containerVolumeMounts: []v1.VolumeMount{{
						ReadOnly:  true,
						Name:      trustStoreVolume,
						MountPath: activeGateCacertsPath,
						SubPath:   k8sCertificateFile,
					}},
					volumes: []v1.Volume{{
						Name: trustStoreVolume,
						VolumeSource: v1.VolumeSource{
							EmptyDir: &v1.EmptyDirVolumeSource{},
						},
					}},
				},
			},
		},
		{
			name: "all capability at once",
			args: args{
				dynakube: &dynatracev1beta1.DynaKube{
					Spec: dynatracev1beta1.DynaKubeSpec{
						ActiveGate: dynatracev1beta1.ActiveGateSpec{
							Capabilities: []dynatracev1beta1.CapabilityDisplayName{
								dynatracev1beta1.KubeMonCapability.DisplayName,
								dynatracev1beta1.MetricsIngestCapability.DisplayName,
								dynatracev1beta1.RoutingCapability.DisplayName,
								dynatracev1beta1.DynatraceApiCapability.DisplayName,
							},
						},
					},
				},
			},
			want: &MultiCapability{
				capabilityBase: capabilityBase{
					enabled:   true,
					shortName: MultiActiveGateName,
					argName: strings.Join([]string{
						dynatracev1beta1.KubeMonCapability.ArgumentName,
						dynatracev1beta1.MetricsIngestCapability.ArgumentName,
						dynatracev1beta1.RoutingCapability.ArgumentName,
						dynatracev1beta1.DynatraceApiCapability.ArgumentName},
						","),
					properties: props,
					Configuration: Configuration{
						SetDnsEntryPoint:     true,
						SetReadinessPort:     true,
						SetCommunicationPort: true,
						CreateService:        true,
						ServiceAccountOwner:  "kubernetes-monitoring",
					},
					initContainersTemplates: []v1.Container{
						{
							Name:            initContainerTemplateName,
							ImagePullPolicy: v1.PullAlways,
							WorkingDir:      k8scrt2jksWorkingDir,
							Command:         []string{"/bin/bash"},
							Args:            []string{"-c", k8scrt2jksPath},
							VolumeMounts: []v1.VolumeMount{
								{
									ReadOnly:  false,
									Name:      trustStoreVolume,
									MountPath: activeGateSslPath,
								},
							},
						},
					},
					containerVolumeMounts: []v1.VolumeMount{{
						ReadOnly:  true,
						Name:      trustStoreVolume,
						MountPath: activeGateCacertsPath,
						SubPath:   k8sCertificateFile,
					}},
					volumes: []v1.Volume{{
						Name: trustStoreVolume,
						VolumeSource: v1.VolumeSource{
							EmptyDir: &v1.EmptyDirVolumeSource{},
						},
					}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMultiCapability(tt.args.dynakube); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMultiCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}
