package capability

import (
	"path/filepath"
	"reflect"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

func Test_capabilityBase_GetProperties(t *testing.T) {
	props := &dynatracev1alpha1.CapabilityProperties{}

	type fields struct {
		properties *dynatracev1alpha1.CapabilityProperties
	}
	tests := []struct {
		name   string
		fields fields
		want   *dynatracev1alpha1.CapabilityProperties
	}{
		{
			name: "properties address is preserved",
			fields: fields{
				properties: props,
			},
			want: &dynatracev1alpha1.CapabilityProperties{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &capabilityBase{
				properties: tt.fields.properties,
			}
			if got := c.GetProperties(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("capabilityBase.GetProperties() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_capabilityBase_GetConfiguration(t *testing.T) {
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
			if got := c.GetConfiguration(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("capabilityBase.GetConfiguration() = %v, want %v", got, tt.want)
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
				moduleName: tt.fields.moduleName,
			}
			if got := c.GetModuleName(); got != tt.want {
				t.Errorf("capabilityBase.GetModuleName() = %v, want %v", got, tt.want)
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
				capabilityName: tt.fields.capabilityName,
			}
			if got := c.GetCapabilityName(); got != tt.want {
				t.Errorf("capabilityBase.GetCapabilityName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateStatefulSetName(t *testing.T) {
	c := NewKubeMonCapability(nil, nil)
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
			want: instanceName + "-" + c.GetModuleName(),
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
	props := &dynatracev1alpha1.CapabilityProperties{}

	type args struct {
		crProperties *dynatracev1alpha1.CapabilityProperties
	}
	tests := []struct {
		name string
		args args
		want *KubeMonCapability
	}{
		{
			name: "",
			args: args{
				crProperties: props,
			},
			want: &KubeMonCapability{
				capabilityBase: capabilityBase{
					moduleName:     "kubemon",
					capabilityName: "kubernetes_monitoring",
					properties:     props,
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
			if got := NewKubeMonCapability(tt.args.crProperties, nil); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewKubeMonCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRoutingCapability(t *testing.T) {
	const tlsSecretName = "tls-secret"
	agSpecWithTls := &dynatracev1alpha1.ActiveGateSpec{
		TlsSecretName: tlsSecretName,
	}

	props := &dynatracev1alpha1.CapabilityProperties{}

	type args struct {
		crProperties *dynatracev1alpha1.CapabilityProperties
		agSpec *dynatracev1alpha1.ActiveGateSpec
	}
	tests := []struct {
		name string
		args args
		want *RoutingCapability
	}{
		{
			name: "default",
			args: args{
				crProperties: props,
				agSpec: nil,
			},
			want: &RoutingCapability{
				capabilityBase: capabilityBase{
					moduleName:     "routing",
					capabilityName: "MSGrouter",
					properties:     props,
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
			name: "with-tls-secert-set",
			args: args{
				crProperties: props,
				agSpec:       agSpecWithTls,
			},
			want: &RoutingCapability{
				capabilityBase: capabilityBase{
					moduleName:     "routing",
					capabilityName: "MSGrouter",
					properties:     props,
					volumes: []v1.Volume{{
						Name: jettyCerts,
						VolumeSource: v1.VolumeSource{
							Secret: &v1.SecretVolumeSource{
								SecretName: agSpecWithTls.TlsSecretName,
							},
						},
					}},
					containerVolumeMounts: []v1.VolumeMount{{
						ReadOnly:  true,
						Name:      jettyCerts,
						MountPath: filepath.Join(secretsRootDir, "tls"),
					}},
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
			if got := NewRoutingCapability(tt.args.crProperties, tt.args.agSpec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRoutingCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewMetricsCapability(t *testing.T) {
	props := &dynatracev1alpha1.CapabilityProperties{}

	type args struct {
		crProperties *dynatracev1alpha1.CapabilityProperties
	}
	tests := []struct {
		name string
		args args
		want *DataIngestCapability
	}{
		{
			name: "",
			args: args{
				crProperties: props,
			},
			want: &DataIngestCapability{
				capabilityBase: capabilityBase{
					moduleName:     "data-ingest",
					capabilityName: "metrics_ingest",
					properties:     props,
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
			if got := NewDataIngestCapability(tt.args.crProperties, nil); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDataIngestCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}
