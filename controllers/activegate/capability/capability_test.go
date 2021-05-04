package capability

import (
	"reflect"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
)

func TestCapability_CalculateStatefulSetName(t *testing.T) {
	type fields struct {
		ModuleName     string
		CapabilityName string
		Properties     *dynatracev1alpha1.CapabilityProperties
		Configuration  Configuration
	}
	type args struct {
		instanceName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "",
			fields: fields{
				ModuleName: "mmm",
			},
			args: args{
				instanceName: "iiii",
			},
			want: "iiii-mmm",
		},
		{
			name: "",
			fields: fields{
				ModuleName: "",
			},
			args: args{
				instanceName: "iiii",
			},
			want: "iiii-",
		},
		{
			name: "",
			fields: fields{
				ModuleName: "mmm",
			},
			args: args{
				instanceName: "",
			},
			want: "-mmm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Capability{
				ModuleName:     tt.fields.ModuleName,
				CapabilityName: tt.fields.CapabilityName,
				Properties:     tt.fields.Properties,
				Configuration:  tt.fields.Configuration,
			}
			if got := c.CalculateStatefulSetName(tt.args.instanceName); got != tt.want {
				t.Errorf("Capability.CalculateStatefulSetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMakeCapapability(t *testing.T) {
	validProperties := &dynatracev1alpha1.CapabilityProperties{}

	type args struct {
		c          CapabilityType
		properties *dynatracev1alpha1.CapabilityProperties
	}
	tests := []struct {
		name string
		args args
		want *Capability
	}{
		{
			name: "kubemon",
			args: args{
				c:          Kubemon,
				properties: validProperties,
			},
			want: &Capability{
				ModuleName:     "kubemon",
				CapabilityName: "kubernetes_monitoring",
				Properties:     validProperties,
				Configuration: Configuration{
					ServiceAccountOwner: "kubernetes-monitoring",
				},
			},
		},
		{
			name: "routing",
			args: args{
				c:          Routing,
				properties: validProperties,
			},
			want: &Capability{
				ModuleName:     "routing",
				CapabilityName: "MSGrouter",
				Properties:     validProperties,
				Configuration: Configuration{
					SetDnsEntryPoint:     true,
					SetReadinessPort:     true,
					SetCommunicationPort: true,
					CreateService:        true,
				},
			},
		},
		{
			name: "mint",
			args: args{
				c:          Mint,
				properties: validProperties,
			},
			want: &Capability{
				ModuleName:     "mint",
				CapabilityName: "metrics_ingest",
				Properties:     validProperties,
				Configuration: Configuration{
					SetDnsEntryPoint:     true,
					SetReadinessPort:     true,
					SetCommunicationPort: true,
					CreateService:        true,
				},
			},
		},
		{
			name: "unknown",
			args: args{
				c:          123123,
				properties: validProperties,
			},
			want: nil,
		},
		{
			name: "properties is nil",
			args: args{
				c:          Kubemon,
				properties: nil,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MakeCapapability(tt.args.c, tt.args.properties); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeCapapability() = %v, want %v", got, tt.want)
			}
		})
	}
}
