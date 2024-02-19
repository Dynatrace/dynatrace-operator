//go:build e2e

package environment

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func GetStandardKubeClusterEnvConfig() *envconf.Config {
	kubeConfigPath := conf.ResolveKubeConfigFile()
	cfg, _ := envconf.NewFromFlags()

	return cfg.WithKubeconfigFile(kubeConfigPath)
}
