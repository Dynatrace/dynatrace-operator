//go:build e2e

package deployersamples

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

var samplesDir = filepath.Join(project.RootDir(), "assets", "samples", "deployer")

func EscalateNoCSI() features.Feature {
	return feature(
		"deployer-clusterrole-no-csi.yaml",
		"dynatrace-deployer-no-csi",
		false,
	)
}

func EscalateWithCSI() features.Feature {
	return feature(
		"deployer-clusterrole-with-csi.yaml",
		"dynatrace-deployer-with-csi",
		true,
	)
}

func NoEscalateNoCSI() features.Feature {
	return feature(
		"deployer-clusterrole-no-escalate-no-csi.yaml",
		"dynatrace-deployer-no-escalate-no-csi",
		false,
	)
}

func NoEscalateWithCSI() features.Feature {
	return feature(
		"deployer-clusterrole-no-escalate-with-csi.yaml",
		"dynatrace-deployer-no-escalate-with-csi",
		true,
	)
}

func feature(clusterRole, serviceAccountName string, withCSI bool) features.Feature {
	builder := features.New("deployer-sample-" + clusterRole)

	clusterRolePath := filepath.Join(samplesDir, clusterRole)
	serviceAccount := "system:serviceaccount:default:" + serviceAccountName

	builder.Setup(installRole(clusterRolePath))

	builder.Assess("Operator installed", assessHelmOp("helm install", serviceAccount, withCSI, helmInstall))

	builder.Assess("Operator upgraded", assessHelmOp("helm upgrade", serviceAccount, withCSI, helmUpgrade))

	builder.Teardown(teardownHelmOp(serviceAccount, clusterRolePath))

	return builder.Feature()
}

func installRole(path string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		ctx, err := manifests.InstallFromFile(path)(ctx, c)
		require.NoError(t, err, "failed to apply %s", path)

		return ctx
	}
}

func assessHelmOp(name, serviceAccount string, withCSI bool, helmFn func(serviceAccount string, withCSI bool) error) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		t.Helper()

		err := helmFn(serviceAccount, withCSI)
		require.NoError(t, err, "%s failed for %q", name, serviceAccount)

		ctx, err = operator.VerifyInstall(ctx, envConfig, withCSI)
		require.NoError(t, err, "operator installation verification failed")

		return ctx
	}
}

func teardownHelmOp(serviceAccount, clusterRolePath string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		if c.FailFast() {
			return ctx
		}

		err := helmUninstall(serviceAccount)
		require.NoError(t, err, "helm uninstall failed")

		ctx, err = manifests.UninstallFromFile(clusterRolePath)(ctx, c)
		require.NoError(t, err, "role cleanup failed")

		return ctx
	}
}

func helmInstall(serviceAccount string, withCSI bool) error {
	return operator.InstallViaHelm(
		"",
		withCSI,
		helm.WithArgs("--install"),
		helm.WithArgs("--create-namespace"),
		helm.WithArgs("--set", "webhook.replicas=1"),
		helm.WithArgs("--kube-as-user", serviceAccount),
	)
}

// helmUpgrade runs helm upgrade (no --install) to cover upgrade-specific behavior,
// e.g. the crd storage migration pre-upgrade hook Job that exercises batch/jobs permissions.
func helmUpgrade(serviceAccount string, withCSI bool) error {
	return operator.InstallViaHelm(
		"",
		withCSI,
		helm.WithArgs("--set", "webhook.replicas=1"),
		helm.WithArgs("--set", "crdStorageMigrationJob=true"),
		helm.WithArgs("--kube-as-user", serviceAccount),
	)
}

func helmUninstall(serviceAccount string) error {
	return operator.UninstallViaHelm(
		"dynatrace-operator",
		"dynatrace",
		helm.WithArgs("--kube-as-user", serviceAccount))
}
