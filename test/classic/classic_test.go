//go:build e2e

package classic

import (
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"testing"
)

const (
	sampleAppsName      = "myapp"
	sampleAppsNamespace = "test-namespace-1"

	dynakubeName       = "dynakube"
	dynatraceNamespace = "dynatrace"

	installSecretsPath = "../testdata/secrets/classic-fullstack-install.yaml"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists())
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.DeleteIfExists(sampleAppsNamespace))
	testEnvironment.BeforeEachTest(namespace.Recreate(dynatraceNamespace))

	//testEnvironment.AfterEachTest(dynakube.DeleteIfExists())
	//testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	//testEnvironment.AfterEachTest(namespace.Delete(sampleAppsNamespace))
	//testEnvironment.AfterEachTest(namespace.Delete(dynatraceNamespace))

	testEnvironment.Run(m)
}

func TestClassicFullStack(t *testing.T) {
	testEnvironment.Test(t, install(t))
}

func getSecretConfig(t *testing.T) secrets.Secret {
	currentWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)

	secretPath := path.Join(currentWorkingDirectory, installSecretsPath)
	secretConfig, err := secrets.NewFromConfig(afero.NewOsFs(), secretPath)

	require.NoError(t, err)

	return secretConfig
}
