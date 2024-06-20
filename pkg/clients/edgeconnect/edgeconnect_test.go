package edgeconnect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeHostMappings(t *testing.T) {
	k8sAutomationHostPattern := "my-edgeconnect.dynatrace.a273ec656-603d-46c8-b5f5-5c47a6903dff.kubernetes-automation"

	type args struct {
		k8sAutomationHostPattern string
	}
	tests := []struct {
		name string
		args args
		want []HostMapping
	}{
		{
			name: "Empty parameter",
			args: args{
				k8sAutomationHostPattern: "",
			},
			want: []HostMapping{},
		},
		{
			name: "Present k8sAutomationHostPattern",
			args: args{
				k8sAutomationHostPattern: k8sAutomationHostPattern,
			},
			want: []HostMapping{
				{From: k8sAutomationHostPattern, To: defaultKubernetesDNS},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeHostMappings(tt.args.k8sAutomationHostPattern)
			require.EqualValues(t, tt.want, got)
		})
	}
}
