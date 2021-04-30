package activegate

import (
	"reflect"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
)

func TestBuildLabels(t *testing.T) {
	type args struct {
		instance             *v1alpha1.DynaKube
		feature              string
		capabilityProperties *v1alpha1.CapabilityProperties
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildLabels(tt.args.instance, tt.args.feature, tt.args.capabilityProperties); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildLabelsFromInstance(t *testing.T) {
	type args struct {
		instance *v1alpha1.DynaKube
		feature  string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildLabelsFromInstance(tt.args.instance, tt.args.feature); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildLabelsFromInstance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeLabels(t *testing.T) {
	type args struct {
		labels []map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeLabels(tt.args.labels...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}
