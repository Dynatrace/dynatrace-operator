//go:build e2e

package deployersamples

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
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
	testDataDir = filepath.Join(project.TestDataDir(), "deployer")

	sharedSAFile = filepath.Join(samplesDir, "deployer-sa-and-binding.yaml")
)

func EscalateWithoutCSIFeature(t *testing.T) features.Feature {
	builder := features.New("deployer-sample-no-csi")

	clusterRole := filepath.Join(samplesDir, "deployer-clusterrole-no-csi.yaml")
	serviceAccount := "system:serviceaccount:dynatrace:dynatrace-deployer-no-csi"

	builder.Setup(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		_, err := manifests.InstallFromFile(clusterRole)(ctx, c)
		require.NoError(t, err, "failed to apply %s", clusterRole)

		return ctx
	})

	builder.Assess("install operator", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		ctx, err := operator.InstallWithHelmAsUser("", false, serviceAccount)(ctx, c)
		require.NoError(t, err, "failed to install %s", serviceAccount)

		return ctx
	})

	// builder.Assess("helm install succeeds as deployer", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	//	t.Helper()
	//
	//	return assessInstallSucceeds(ctx, t, serviceAccount, csiEnabled)
	//})
	//builder.Assess("deployer can upgrade existing release", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	//	t.Helper()
	//
	//	return assessUpgrade(ctx, t, serviceAccount, csiEnabled)
	//})
	//builder.Assess("deployer can manage DynaKube and EdgeConnect CRs", assessCRLifecycle(serviceAccount))
	//
	//builder.Teardown(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	//	t.Helper()
	//
	//	helmUninstall(t, serviceAccount)
	//
	//	_, err := manifests.UninstallFromFile(clusterRole)(ctx, c)
	//	assert.NoError(t, err)
	//
	//	return ctx
	//})

	return builder.Feature()
}

// func EscalateNoCsi(t *testing.T) features.Feature {
//	return Feature(t,
//		filepath.Join(samplesDir, "deployer-clusterrole-no-csi.yaml"),
//		"system:serviceaccount:dynatrace:dynatrace-deployer-no-csi",
//		false, false,
//	)
//}
//
//func EscalateWithCsi(t *testing.T) features.Feature {
//	return Feature(t,
//		filepath.Join(samplesDir, "deployer-clusterrole-with-csi.yaml"),
//		"system:serviceaccount:dynatrace:dynatrace-deployer-with-csi",
//		true, false,
//	)
//}
//
//func NoEscalateNoCsi(t *testing.T) features.Feature {
//	return Feature(t,
//		filepath.Join(samplesDir, "deployer-clusterrole-no-escalate-no-csi.yaml"),
//		"system:serviceaccount:dynatrace:dynatrace-deployer-no-escalate-no-csi",
//		false, false,
//	)
//}
//
//func NoEscalateWithCsi(t *testing.T) features.Feature {
//	return Feature(t,
//		filepath.Join(samplesDir, "deployer-clusterrole-no-escalate-with-csi.yaml"),
//		"system:serviceaccount:dynatrace:dynatrace-deployer-no-escalate-with-csi",
//		true, false,
//	)
//}
//
//func InsufficientPermissions(t *testing.T) features.Feature {
//	return Feature(t,
//		filepath.Join(testDataDir, "insufficient-clusterrole-permissions.yaml"),
//		"system:serviceaccount:default:dynatrace-deployer-insufficient",
//		false, true,
//	)
//}
//
//func assessInstallSucceeds(ctx context.Context, t *testing.T, serviceAccount string, csiEnabled bool) context.Context {
//	t.Helper()
//
//	err := helmInstall(serviceAccount, csiEnabled)
//	require.NoError(t, err, "helm install as %q should succeed but failed", serviceAccount)
//
//	return ctx
//}
//
//func assessInstallFails(ctx context.Context, t *testing.T, serviceAccount string, csiEnabled bool) context.Context {
//	t.Helper()
//
//	err := helmInstall(serviceAccount, csiEnabled)
//	require.Error(t, err, "helm install as %q should have failed but succeeded", serviceAccount)
//
//	// Verify it's a permission error, not some other issue
//	assert.Contains(t, err.Error(), "forbidden", "expected a permission denied error")
//
//	return ctx
//}

func assessUpgrade(ctx context.Context, t *testing.T, serviceAccount string, csiEnabled bool) context.Context {
	t.Helper()

	err := helmUpgrade(serviceAccount, csiEnabled)
	require.NoError(t, err, "helm upgrade as %q should succeed but failed", serviceAccount)

	return ctx
}

// helmUpgrade runs helm upgrade (no --install) to cover upgrade-specific behavior,
// e.g. the crd storage migration pre-upgrade hook Job that exercises batch/jobs permissions.
func helmUpgrade(serviceAccount string, csiEnabled bool) error {
	return operator.InstallViaHelm("", csiEnabled,
		helm.WithArgs("--set", "webhook.replicas=1"),
		helm.WithArgs("--set", "crdStorageMigrationJob=true"),
		helm.WithArgs("--kube-as-user", serviceAccount),
	)
}

func helmUninstall(t *testing.T, serviceAccount string) {
	t.Helper()

	manager := helm.New("''")

	// Try with impersonation first
	_ = manager.RunUninstall(
		helm.WithReleaseName(releaseName),
		helm.WithNamespace(targetNamespace),
		helm.WithArgs("--kube-as-user", serviceAccount),
	)
	// Also try without impersonation in case the SA can't uninstall
	_ = manager.RunUninstall(
		helm.WithReleaseName(releaseName),
		helm.WithNamespace(targetNamespace),
	)
}

// assessCRLifecycle verifies that the deployer SA can perform the full ArgoCD-style
// lifecycle (create, get, update, delete) on DynaKube and EdgeConnect custom resources.
func assessCRLifecycle(serviceAccount string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := []string{"dynakubes.dynatrace.com", "edgeconnects.dynatrace.com"}
		verbs := []string{"create", "get", "update", "delete", "list"}
		for _, resource := range resources {
			for _, verb := range verbs {
				sar := newSubjectAccessReview(targetNamespace, serviceAccount, verb, resource)
				require.NoError(t, envConfig.Client().Resources().Create(ctx, sar))
				assert.True(t, sar.Status.Allowed, "expected %q verb to be allowed on %q resource for %q, but it was denied: %s", verb, resource, serviceAccount, sar.Status.Reason)
			}
		}

		return ctx
	}
}

func newSubjectAccessReview(namespace, serviceAccount, verb, resource string) *authorizationv1.SubjectAccessReview {
	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User: serviceAccount,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Resource:  resource,
			},
		},
	}

	return sar
}
