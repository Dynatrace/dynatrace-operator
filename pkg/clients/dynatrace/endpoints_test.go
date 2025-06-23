package dynatrace

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
)

func Test_dynatraceClient_getOneAgentConnectionInfoUrl(t *testing.T) {
	tests := []struct {
		name        string
		networkZone string
		want        string
	}{
		{
			name:        "with network zone",
			networkZone: "mynetworkzone",
			want:        "https://testenvironment.live.dynatrace.com/api/v1/deployment/installer/agent/connectioninfo?networkZone=mynetworkzone&defaultZoneFallback=true",
		},
		{
			name:        "without network zone",
			networkZone: "",
			want:        "https://testenvironment.live.dynatrace.com/api/v1/deployment/installer/agent/connectioninfo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.NewClient()

			dtc := &dynatraceClient{
				url:         "https://testenvironment.live.dynatrace.com/api",
				networkZone: tt.networkZone,
			}
			if got := dtc.getOneAgentConnectionInfoURL(); got != tt.want {
				t.Errorf("dynatraceClient.getOneAgentConnectionInfoURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dynatraceClient_getActiveGateConnectionInfoUrl(t *testing.T) {
	tests := []struct {
		name        string
		networkZone string
		want        string
	}{
		{
			name:        "with network zone",
			networkZone: "mynetworkzone",
			want:        "https://testenvironment.live.dynatrace.com/api/v1/deployment/installer/gateway/connectioninfo",
		},
		{
			name:        "without network zone",
			networkZone: "",
			want:        "https://testenvironment.live.dynatrace.com/api/v1/deployment/installer/gateway/connectioninfo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.NewClient()

			dtc := &dynatraceClient{
				url:         "https://testenvironment.live.dynatrace.com/api",
				networkZone: tt.networkZone,
			}
			if got := dtc.getActiveGateConnectionInfoURL(); got != tt.want {
				t.Errorf("dynatraceClient.getActiveGateConnectionInfoURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
