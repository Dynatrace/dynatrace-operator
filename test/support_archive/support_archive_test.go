//go:build e2e

package support_archive

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"os"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return
	}
	//	require.NoError(m, err)

	fmt.Println(workingDir)

	testEnvironment = environment.Get()
	//	testEnvironment.BeforeEachTest(namespace.DeleteIfExists(sampleapps.Namespace))
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.Recreate(dynakube.Namespace))

	//	testEnvironment.AfterEachTest(namespace.Delete(sampleapps.Namespace))
	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.Namespace))

	testEnvironment.Run(m)
}

func TestSupportArchive(t *testing.T) {
	testEnvironment.Test(t, installOperatorAndDynakube(t))
}

// Note: mainly for dev purposes, test requires a running cluster with deployed operator to be successful
func TestExec(t *testing.T) {
	t.Skip("helper for development")

	kubeConfigPath := conf.ResolveKubeConfigFile()
	envConfig := envconf.NewWithKubeConfig(kubeConfigPath)
	executeTroubleshoot(context.TODO(), t, envConfig)
}
