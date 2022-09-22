package cluster_intel_collector

import (
	"testing"

	cmdConfig "github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func BenchmarkLogCollector(b *testing.B) {
	logCollectorCmd := NewCicCommandBuilder().SetConfigProvider(cmdConfig.NewKubeConfigProvider()).Build()
	logCollectorCmd.Execute()
}
