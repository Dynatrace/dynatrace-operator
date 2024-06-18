package edgeconnect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeHostMappings(t *testing.T) {
	hostPattern1 := "my-edgeconnect.dynatrace.a273ec656-603d-46c8-b5f5-5c47a6903dff.kubernetes-automation"
	hostPattern2 := "super-edgeconnect.dynatrace.a273ec656-603d-46c8-b5f5-5c47a6903dfd.kubernetes-automation"

	type args struct {
		hostPatterns []string
	}
	tests := []struct {
		name string
		args args
		want []HostMapping
	}{
		{
			name: "No host patterns",
			args: args{
				hostPatterns: []string{},
			},
			want: []HostMapping{},
		},
		{
			name: "Single host pattern",
			args: args{
				hostPatterns: []string{hostPattern1},
			},
			want: []HostMapping{
				{From: hostPattern1, To: defaultKubernetesDns},
			},
		},
		{
			name: "Multiple host patterns",
			args: args{
				hostPatterns: []string{hostPattern1, hostPattern2},
			},
			want: []HostMapping{
				{From: hostPattern1, To: defaultKubernetesDns},
				{From: hostPattern2, To: defaultKubernetesDns},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeHostMappings(tt.args.hostPatterns)
			require.EqualValues(t, tt.want, got)
		})
	}
}
