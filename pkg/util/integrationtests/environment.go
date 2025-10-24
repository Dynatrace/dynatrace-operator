package integrationtests

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest" //nolint:revive
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/projectpath"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	testEnv *envtest.Environment
)

func SetupTestEnvironment(t *testing.T) client.Client {
	setupBaseTestEnv(t)

	testEnv.AttachControlPlaneOutput = true

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

func SetupWebhookTestEnvironment(t *testing.T, webhookOptions envtest.WebhookInstallOptions, webhookSetup func(ctrl.Manager) error) client.Client {
	setupBaseTestEnv(t)

	testEnv.WebhookInstallOptions = webhookOptions

	// start test environment
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err, "start environment")
	}

	t.Cleanup(func() {
		err := testEnv.Stop()
		if err != nil {
			// test is already ending, no need to explicitly fail test
			t.Error(err, "stop env")
		}
	})

	clt, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Fatal(err, "new client")
	}

	// start webhook server using Manager.
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
	})
	if err != nil {
		t.Fatal(err, "new manager")
	}

	if err := webhookSetup(mgr); err != nil {
		t.Fatal(err, "webhook setup")
	}

	// this code is adapted from the ginkgo/gomega boilerplate that kubebuilder generates

	go func() {
		if err := mgr.Start(t.Context()); err != nil {
			// don't call t.Fatal in a goroutine
			t.Error(err, "run manager")
		}
	}()

	// wait for the webhook server to get ready.
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)

	waitCh := make(chan struct{})
	defer close(waitCh)

	err = wait.PollUntilContextTimeout(
		t.Context(),
		500*time.Millisecond, 10*time.Second,
		false,
		func(ctx context.Context) (bool, error) {
			conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
			if err != nil {
				return false, err
			}

			return true, conn.Close()
		},
	)
	if err != nil {
		t.Fatal(err, "wait for webhook")
	}

	return clt
}

func setupBaseTestEnv(t *testing.T) {
	t.Helper()

	// specify test environment configuration
	testEnv = &envtest.Environment{
		Scheme:                scheme.Scheme,
		CRDDirectoryPaths:     []string{filepath.Join(projectpath.Root, "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	if err := addScheme(testEnv); err != nil {
		t.Fatal(err)
	}

	logf.SetLogger(logd.Get().Logger)
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

	err = v1beta5.AddToScheme(testEnv.Scheme)
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
