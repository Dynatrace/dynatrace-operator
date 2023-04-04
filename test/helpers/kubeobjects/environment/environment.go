//go:build e2e

package environment

import (
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

const (
	useKind = "TEST_ENV_USE_KIND"
)

func Get() env.Environment {
	if os.Getenv(useKind) == "true" {
		environment := env.New()
		environment.Setup(envfuncs.CreateKindCluster(envconf.RandomName("operator-test", 10)))
		return environment
	}

	kubeConfigPath := conf.ResolveKubeConfigFile()

	cfg, _ := envconf.NewFromFlags()
	envConfig := cfg.WithKubeconfigFile(kubeConfigPath)
	return env.NewWithConfig(envConfig)
}
