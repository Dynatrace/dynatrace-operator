package integrationtests

import (
	"os"
	"path/filepath"
	"testing"

	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest" //nolint:revive
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/projectpath"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	testEnv *envtest.Environment
)

func SetupTestEnvironment(t *testing.T) client.Client {
	// specify test environment configuration
	testEnv = &envtest.Environment{
		Scheme:                   scheme.Scheme,
		CRDDirectoryPaths:        []string{filepath.Join(projectpath.Root, "config", "crd", "bases")},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// TODO: refactor and reuse from e2e tests or another place
	if err := addScheme(testEnv); err != nil {
		t.Fatal(err)
	}

	// start test environment
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}

	clt, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Fatal(err)
	}

	return clt
}

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join(projectpath.Root, "bin", "k8s")

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}

	return ""
}

func addScheme(testEnv *envtest.Environment) error {
	err := latest.AddToScheme(testEnv.Scheme)
	if err != nil {
		return err
	}

	err = v1beta4.AddToScheme(testEnv.Scheme)
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
