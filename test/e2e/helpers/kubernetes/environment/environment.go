//go:build e2e

package environment

import (
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	skipScalingEnvVar   = "E2E_SKIP_SCALING"
	scalingFeatureRegex = ".*(hpa|enforce-replicas)"
)

func GetStandardKubeClusterEnvConfig() *envconf.Config {
	kubeConfigPath := conf.ResolveKubeConfigFile()
	cfg, _ := envconf.NewFromFlags()
	cfg = cfg.WithKubeconfigFile(kubeConfigPath)

	if os.Getenv(skipScalingEnvVar) == "true" {
		cfg.WithSkipFeatureRegex(scalingFeatureRegex)
	}

	return cfg
}
