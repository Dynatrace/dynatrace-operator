//go:build integrationtests

package integrationtests

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"testing"
)

var (
	testEnv *envtest.Environment
)

func SetupTestEnvironment(t *testing.T) client.Client {
	// specify test environment configuration
	os.Setenv("KUBEBUILDER_ASSETS", "/home/adamo/workspace/stargate/dynatrace-operator/testbin/bin/k8s/1.31.0-linux-amd64")

	testEnv = &envtest.Environment{
		Scheme:            scheme.Scheme,
		CRDDirectoryPaths: []string{"./config/deploy/kubernetes"},
	}

	if err := addScheme(testEnv); err != nil {
		t.Fatal(err)
	}

	// start test environment
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}

	clt, err := client.New(cfg, client.Options{}) //, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatal(err)
	}

	return clt
}

func addScheme(testEnv *envtest.Environment) error {
	err := v1beta4.AddToScheme(testEnv.Scheme)
	if err != nil {
		return err
	}

	err = v1beta3.AddToScheme(testEnv.Scheme)
	if err != nil {
		return err
	}

	err = v1beta2.AddToScheme(testEnv.Scheme)
	if err != nil {
		return err
	}

	err = v1beta1.AddToScheme(testEnv.Scheme)
	if err != nil {
		return err
	}

	err = v1alpha2.AddToScheme(testEnv.Scheme)
	if err != nil {
		return err
	}

	err = v1alpha1.AddToScheme(testEnv.Scheme)
	if err != nil {
		return err
	}

	return nil
}

func DestroyTestEnvironment(t *testing.T) {
	// stop test environment
	err := testEnv.Stop()
	if err != nil {
		t.Fatal(err)
	}
}
