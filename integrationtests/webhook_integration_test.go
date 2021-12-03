//go:build integration
// +build integration

package integrationtests

import (
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func setupTestEnvironment(_ *testing.T) *envtest.Environment {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "default", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths:                    []string{filepath.Join("..", "config", "common", "webhook", "validation")},
			IgnoreErrorIfPathMissing: true,
		},
	}
	return testEnv
}

func TestAddDynakubeValidationWebhookToManager(t *testing.T) {
	testEnv := setupTestEnvironment(t)
	cfg, err := testEnv.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	require.NoError(t, err)
	require.NotNil(t, mgr)

	err = validation.AddDynakubeValidationWebhookToManager(mgr)
	assert.NoError(t, err)
}
