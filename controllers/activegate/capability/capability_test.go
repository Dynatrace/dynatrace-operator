package capability

import (
	"reflect"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
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
	cap := NewKubeMonCapability(nil)
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
				capability:   cap,
				instanceName: instanceName,
			},
			want: instanceName + "-" + cap.GetModuleName(),
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
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewKubeMonCapability(tt.args.crProperties); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewKubeMonCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRoutingCapability(t *testing.T) {
	props := &dynatracev1alpha1.CapabilityProperties{}

	type args struct {
		crProperties *dynatracev1alpha1.CapabilityProperties
	}
	tests := []struct {
		name string
		args args
		want *RoutingCapability
	}{
		{
			name: "",
			args: args{
				crProperties: props,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRoutingCapability(tt.args.crProperties); !reflect.DeepEqual(got, tt.want) {
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
		want *MetricsCapability
	}{
		{
			name: "",
			args: args{
				crProperties: props,
			},
			want: &MetricsCapability{
				capabilityBase: capabilityBase{
					moduleName:     "metrics",
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
			if got := NewMetricsCapability(tt.args.crProperties); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMetricsCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}
