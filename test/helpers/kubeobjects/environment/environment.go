//go:build e2e

package environment

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func GetStandardKubeClusterEnvironment() env.Environment {
	kubeConfigPath := conf.ResolveKubeConfigFile()
	cfg, _ := envconf.NewFromFlags()
	envConfig := cfg.WithKubeconfigFile(kubeConfigPath)
	return env.NewWithConfig(envConfig)
}
