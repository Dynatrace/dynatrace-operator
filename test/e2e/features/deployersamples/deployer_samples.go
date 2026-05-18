//go:build e2e

package deployersamples

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

const (
	targetNamespace = "dynatrace"
	releaseName     = "dynatrace-operator"
)

var (
	samplesDir  = filepath.Join(project.RootDir(), "assets", "samples", "deployer")
	testDataDir = filepath.Join(project.TestDataDir(), "deployer-samples")

	sharedSAFile = filepath.Join(samplesDir, "deployer-sa-and-binding.yaml")
)

// sharedFile wraps install/uninstall for the shared SA and binding manifest.
type sharedFile struct {
	path string
}

// SharedSAFile returns a handle to install/uninstall the shared service accounts and bindings.
func SharedSAFile() *sharedFile {
	return &sharedFile{path: sharedSAFile}
}

func (s *sharedFile) Install(ctx context.Context, c *envconf.Config) (context.Context, error) {
	return manifests.InstallFromFile(s.path)(ctx, c)
}

func (s *sharedFile) Uninstall(ctx context.Context, c *envconf.Config) (context.Context, error) {
	return manifests.UninstallFromFile(s.path)(ctx, c)
}

type variant struct {
	name           string
	clusterRole    string
	serviceAccount string
	csiEnabled     bool
	expectFailure  bool
}

var positiveVariants = []variant{
	{
		name:           "escalate-no-csi",
		clusterRole:    filepath.Join(samplesDir, "deployer-clusterrole-no-csi.yaml"),
		serviceAccount: "system:serviceaccount:dynatrace:dynatrace-deployer-no-csi",
		csiEnabled:     false,
	},
	{
		name:           "escalate-with-csi",
		clusterRole:    filepath.Join(samplesDir, "deployer-clusterrole-with-csi.yaml"),
		serviceAccount: "system:serviceaccount:dynatrace:dynatrace-deployer-with-csi",
		csiEnabled:     true,
	},
	{
		name:           "no-escalate-no-csi",
		clusterRole:    filepath.Join(samplesDir, "deployer-clusterrole-no-escalate-no-csi.yaml"),
		serviceAccount: "system:serviceaccount:dynatrace:dynatrace-deployer-no-escalate-no-csi",
		csiEnabled:     false,
	},
	{
		name:           "no-escalate-with-csi",
		clusterRole:    filepath.Join(samplesDir, "deployer-clusterrole-no-escalate-with-csi.yaml"),
		serviceAccount: "system:serviceaccount:dynatrace:dynatrace-deployer-no-escalate-with-csi",
		csiEnabled:     true,
	},
}

var negativeVariant = variant{
	name:           "insufficient-permissions",
	clusterRole:    filepath.Join(testDataDir, "insufficient-permissions.yaml"),
	serviceAccount: "system:serviceaccount:default:dynatrace-deployer-insufficient",
	csiEnabled:     false,
	expectFailure:  true,
}

// Feature creates a test feature for a specific deployer variant.
func Feature(t *testing.T, v variant) features.Feature {
	builder := features.New("deployer-sample-" + v.name)

	builder.Setup(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		_, err := manifests.InstallFromFile(v.clusterRole)(ctx, c)
		require.NoError(t, err, "failed to apply %s", v.clusterRole)

		return ctx
	})

	if v.expectFailure {
		builder.Assess("helm install fails with insufficient permissions", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			t.Helper()

			return assessInstallFails(ctx, t, v)
		})
	} else {
		builder.Assess("helm install succeeds as deployer", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			t.Helper()

			return assessInstallSucceeds(ctx, t, v)
		})
		builder.Assess("deployer can create and delete DynaKube CR", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			t.Helper()

			return assessDynaKubeLifecycle(ctx, t, v)
		})
	}

	builder.Teardown(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		// Always attempt uninstall (may already be cleaned up on failure path)
		helmUninstall(t)

		_, err := manifests.UninstallFromFile(v.clusterRole)(ctx, c)
		assert.NoError(t, err)

		return ctx
	})

	return builder.Feature()
}

// AllFeatures returns all positive deployer sample features.
func AllFeatures(t *testing.T) []features.Feature {
	feats := make([]features.Feature, 0, len(positiveVariants))
	for _, v := range positiveVariants {
		feats = append(feats, Feature(t, v))
	}

	return feats
}

// NegativeFeature returns the insufficient-permissions feature (manages its own SA).
func NegativeFeature(t *testing.T) features.Feature {
	return Feature(t, negativeVariant)
}

func assessInstallSucceeds(ctx context.Context, t *testing.T, v variant) context.Context {
	t.Helper()

	err := helmInstall(v)
	require.NoError(t, err, "helm install as %q should succeed but failed", v.serviceAccount)

	return ctx
}

func assessInstallFails(ctx context.Context, t *testing.T, v variant) context.Context {
	t.Helper()

	err := helmInstall(v)
	require.Error(t, err, "helm install as %q should have failed but succeeded", v.serviceAccount)

	// Verify it's a permission error, not some other issue
	assert.Contains(t, err.Error(), "forbidden", "expected a permission denied error")

	return ctx
}

func helmInstall(v variant) error {
	return operator.InstallViaHelm("", v.csiEnabled,
		helm.WithArgs("--kube-as-user", v.serviceAccount),
	)
}

func helmUninstall(t *testing.T) {
	t.Helper()

	err := operator.UninstallViaHelm(
		helm.WithReleaseName(releaseName),
		helm.WithNamespace(targetNamespace),
	)
	if err != nil {
		t.Logf("helm uninstall warning (may not have been installed): %v", err)
	}
}

// dynakubeManifestPath returns the path to a temp DynaKube manifest patched
// with the real tenant API URL read from the standard single-tenant secrets file.
func dynakubeManifestPath(t *testing.T) string {
	t.Helper()

	secretConfig := tenant.GetSingleTenantSecret(t)

	staticPath := filepath.Join(testDataDir, "dynakube.yaml")

	content, err := os.ReadFile(staticPath)
	require.NoError(t, err)

	patched := strings.ReplaceAll(string(content), "https://placeholder.live.dynatrace.com/api", secretConfig.APIURL)

	tmp, err := os.CreateTemp("", "dynakube-*.yaml")
	require.NoError(t, err)

	t.Cleanup(func() { os.Remove(tmp.Name()) })

	_, err = tmp.WriteString(patched)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	return tmp.Name()
}

// assessDynaKubeLifecycle verifies that the deployer SA can create and delete
// the DynaKube CR by performing real operations on the cluster.
func assessDynaKubeLifecycle(ctx context.Context, t *testing.T, v variant) context.Context {
	t.Helper()

	// Wait for the webhook deployment to be available before creating the CR —
	// helm install returns before pods are ready and the validating webhook
	// will reject the request if there are no endpoints yet.
	cmd := exec.Command("kubectl", "rollout", "status", "deployment/dynatrace-webhook",
		"--namespace", targetNamespace, "--timeout=120s")
	if out, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, "timed out waiting for dynatrace-webhook: %s", strings.TrimSpace(string(out)))
	}

	manifest := dynakubeManifestPath(t)

	// Register cleanup before creating, so deletion happens even if the assertion fails.
	t.Cleanup(func() {
		_, _ = kubectlAs(v.serviceAccount, "delete", "-f", manifest, "--ignore-not-found")
	})

	out, err := kubectlAs(v.serviceAccount, "apply", "-f", manifest)
	require.NoError(t, err, "deployer %q should be able to create DynaKube CR: %s", v.serviceAccount, out)

	out, err = kubectlAs(v.serviceAccount, "delete", "-f", manifest, "--ignore-not-found")
	assert.NoError(t, err, "deployer %q should be able to delete DynaKube CR: %s", v.serviceAccount, out)

	return ctx
}

// kubectlAs runs a kubectl command impersonating the given service account.
func kubectlAs(sa string, args ...string) (string, error) {
	cmdArgs := append(append([]string(nil), args...), "--as", sa)
	cmd := exec.Command("kubectl", cmdArgs...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}

	return string(out), nil
}
