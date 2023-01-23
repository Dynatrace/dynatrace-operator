//go:build e2e

package environment

import (
	"context"
	"os"
	"testing"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	useKindEnvVar = "TEST_ENV_USE_KIND"
	cleanupEnvVar = "TEST_ENV_CLEANUP"
)

type CleanupMode string

const (
	cleanupAlways      CleanupMode = "always"
	cleanupOnSuccess               = "onSuccess"
	cleanupNever                   = "never"
	cleanupModeDefault             = cleanupAlways
)

type Environment struct {
	env.Environment
	skipCleanup CleanupMode
	t           TestingT
}

type CleanupFunction func(ctx context.Context, envconf *envconf.Config, t *testing.T) (context.Context, error)

func Get() *Environment {
	if os.Getenv(useKindEnvVar) == "true" {
		environment := env.New()
		environment.Setup(envfuncs.CreateKindCluster(envconf.RandomName("operator-test", 10)))

		return &Environment{
			Environment: environment,
			skipCleanup: getSkipCleanupMode(),
		}
	}

	kubeConfigPath := conf.ResolveKubeConfigFile()
	envConfig := envconf.NewWithKubeConfig(kubeConfigPath)

	return &Environment{
		Environment: env.NewWithConfig(envConfig),
		skipCleanup: getSkipCleanupMode(),
	}
}

type TestingT interface {
	Failed() bool
	Errorf(format string, args ...interface{})
}

func (environment *Environment) Test(t TestingT, features ...features.Feature) {
	environment.t = t

	testingT, ok := t.(*testing.T)
	if !ok {
		t.Errorf("Wrong argument type passed, must be *testing.T")
		return
	}

	environment.Environment.Test(testingT, features...)
}

func (environment *Environment) AfterEachTest(cleanupFunctions ...func(context.Context, *envconf.Config, *testing.T) (context.Context, error)) *Environment {
	for _, cleanupFunction := range cleanupFunctions {
		environment.Environment.AfterEachTest(environment.wrapCleanupFunction(cleanupFunction))
	}
	return environment
}

func (environment *Environment) wrapCleanupFunction(cleanupFunction CleanupFunction) func(ctx context.Context, envconf *envconf.Config, t *testing.T) (context.Context, error) {
	wrapper := cleanupFunctionWrapper{
		environment:     environment,
		wrappedFunction: cleanupFunction,
	}
	return wrapper.cleanup
}

func (environment *Environment) shallCleanup() bool {
	switch environment.skipCleanup {
	case cleanupAlways:
		return true
	case cleanupNever:
		return false
	case cleanupOnSuccess:
		return !environment.t.Failed()
	default:
		return true
	}
}

func getSkipCleanupMode() CleanupMode {
	switch os.Getenv(cleanupEnvVar) {
	case string(cleanupAlways):
		return cleanupAlways
	case string(cleanupNever):
		return cleanupNever
	case string(cleanupOnSuccess):
		return cleanupOnSuccess
	default:
		return cleanupModeDefault
	}
}

type cleanupFunctionWrapper struct {
	environment     *Environment
	wrappedFunction func(context.Context, *envconf.Config, *testing.T) (context.Context, error)
}

func (wrapper cleanupFunctionWrapper) cleanup(ctx context.Context, envconf *envconf.Config, t *testing.T) (context.Context, error) {
	if wrapper.environment.shallCleanup() {
		return wrapper.wrappedFunction(ctx, envconf, t)
	}
	return ctx, nil
}
