//go:build e2e

package classic

import (
	"os"
	"path"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

const (
	installSecretsPath = "../testdata/secrets/classic-fullstack-install.yaml"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists())
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.DeleteIfExists(sampleapps.Namespace))
	testEnvironment.BeforeEachTest(namespace.Recreate(dynakube.Namespace))

	testEnvironment.AfterEachTest(dynakube.DeleteIfExists())
	testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.AfterEachTest(namespace.Delete(sampleapps.Namespace))
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.Namespace))

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
