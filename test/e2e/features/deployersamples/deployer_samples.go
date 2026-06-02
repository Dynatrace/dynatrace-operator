//go:build e2e

package deployersamples

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
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
		builder.Assess("deployer can manage DynaKube and EdgeConnect CRs", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			t.Helper()

			return assessCRLifecycle(ctx, t, v)
		})
	}

	builder.Teardown(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		// Always attempt uninstall (may already be cleaned up on failure path)
		helmUninstall(t, v)

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
	chartPath := filepath.Join(project.RootDir(), "config", "helm", "chart", "default")

	// Resolve the platform from the cluster so the chart renders the right resource set.
	// On OpenShift this renders the SCC / *.openshift.io resources, which is what
	// exercises the OpenShift-specific rules in the deployer clusterrole samples.
	plat, err := platform.NewResolver().GetPlatform()
	if err != nil {
		return fmt.Errorf("failed to resolve platform: %w", err)
	}

	args := []string{
		"upgrade", "--install", releaseName, chartPath,
		"--namespace", targetNamespace,
		"--create-namespace",
		"--set", "installCRD=true",
		"--set", fmt.Sprintf("csidriver.enabled=%t", v.csiEnabled),
		"--set", fmt.Sprintf("platform=%s", plat),
		"--set", "manifests=true",
		"--set", "webhook.replicas=1",
		"--kube-as-user", v.serviceAccount,
	}

	cmd := exec.Command("helm", args...)
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}

	return nil
}

func helmUninstall(t *testing.T, v variant) {
	t.Helper()

	args := []string{
		"uninstall", releaseName,
		"--namespace", targetNamespace,
		"--kube-as-user", v.serviceAccount,
	}

	cmd := exec.Command("helm", args...)
	// Ignore errors — may not have been installed (failure case)
	_ = cmd.Run()

	// Also try without impersonation in case the SA can't uninstall
	args = []string{
		"uninstall", releaseName,
		"--namespace", targetNamespace,
	}

	cmd = exec.Command("helm", args...)
	_ = cmd.Run()
}

// assessCRLifecycle verifies that the deployer SA can perform the full ArgoCD-style
// lifecycle (create, get, update, delete) on DynaKube and EdgeConnect custom resources.
func assessCRLifecycle(ctx context.Context, t *testing.T, v variant) context.Context {
	t.Helper()

	resources := []string{"dynakubes.dynatrace.com", "edgeconnects.dynatrace.com"}
	verbs := []string{"create", "get", "update", "delete", "list"}

	for _, resource := range resources {
		for _, verb := range verbs {
			args := []string{
				"auth", "can-i", verb, resource,
				"--namespace", targetNamespace,
				"--as", v.serviceAccount,
			}

			cmd := exec.Command("kubectl", args...)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "deployer %q cannot %s %s: %s", v.serviceAccount, verb, resource, strings.TrimSpace(string(out)))
		}
	}

	return ctx
}
