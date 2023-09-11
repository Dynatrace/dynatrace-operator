package experiment

import (
	"context"
	"os"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"testing"
)

var testenv env.Environment

func TestMain(m *testing.M) {
	testenv = env.New()
	namespace := envconf.RandomName("sample-ns", 16)

	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	testenv = env.NewWithConfig(cfg)

	testenv.Setup(
		envfuncs.CreateNamespace(namespace),
		func(ctx context.Context, config *envconf.Config) (context.Context, error) {
			return ctx, nil
		},
	)
	testenv.Finish(
		envfuncs.DeleteNamespace(namespace),
	)

	os.Exit(testenv.Run(m))
}
