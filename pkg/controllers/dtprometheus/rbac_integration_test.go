// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/projectpath"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// envtest talks to the API server as a member of system:masters, so every RBAC gap is invisible to
// the tests above: they would pass even if the operator's chart roles granted nothing at all. That
// is exactly how ICP-8196 shipped a reconcile path that only failed on a real cluster.
//
// testOperatorRBAC closes that hole without needing a cluster: it grants a test ServiceAccount
// precisely the namespaced rules the helm chart gives the operator, impersonates that
// ServiceAccount, and runs the full reconcile through it. Anything the controller does that the
// chart does not permit surfaces as a Forbidden error here.
//
// Only the namespaced Roles are modeled. The DTPrometheus reconcile is namespace scoped: it
// reads a DTPrometheus and a DynaKube, reads the token Secret, and writes ConfigMaps, Services, a
// StatefulSet and two Deployments, all in the DTPrometheus namespace.
func testOperatorRBAC(t *testing.T, clt client.Client, cfg *rest.Config) {
	const serviceAccountName = "dynatrace-operator"

	f := newFixture(t, clt, "rbac")

	rules := chartRoleRules(t, "operator/role-operator.yaml", "dynatrace-operator")
	rules = append(rules, chartRoleRules(t, "prometheus/role-dtprometheus-operator.yaml", "dynatrace-dtprometheus-operator")...)
	require.NotEmpty(t, rules)

	grantInNamespace(t, clt, f.ns, serviceAccountName, rules)

	impersonating := impersonatingClient(t, cfg, f.ns, serviceAccountName)
	waitForGrant(t, impersonating, f)

	r := newStubbedReconciler(t, impersonating)

	_, err := r.Reconcile(t.Context(), f.request())
	require.NoError(t, err, "the helm chart does not grant the operator everything the reconcile needs")
	assert.False(t, k8serrors.IsForbidden(err))

	// A reconcile that silently skipped work would also produce no error, so confirm the
	// impersonated run actually created everything.
	assert.ElementsMatch(t, expectedManagedObjects(f.dtp), f.listManagedObjects(t))

	t.Run("a second reconcile stays permitted", func(t *testing.T) {
		// Updates need different verbs than creates, and the first reconcile only exercised creates.
		f.dtp.Spec.Gateway.Replicas = new(int32(2))
		f.updateDTPrometheus(t)

		_, err := r.Reconcile(t.Context(), f.request())
		require.NoError(t, err)
		assert.Equal(t, new(int32(2)), f.gatewayStatefulSet(t).Spec.Replicas)
	})

	t.Run("the grant is tight enough to be meaningful", func(t *testing.T) {
		// Guards against the test passing because impersonation is not enforced at all: something
		// the chart deliberately does not grant must be rejected.
		err := impersonating.Create(t.Context(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "rbac-should-not-be-creatable"},
		})
		require.Error(t, err)
		assert.True(t, k8serrors.IsForbidden(err), "expected Forbidden, got %v", err)
	})
}

// waitForGrant blocks until the API server's authorizer has picked up the freshly created
// RoleBinding. The binding is written through the same API server, but the authorizer sees it via
// an informer, so a reconcile started immediately can still be rejected.
func waitForGrant(t *testing.T, impersonating client.Client, f *fixture) {
	t.Helper()

	require.Eventually(t, func() bool {
		return impersonating.Get(t.Context(), client.ObjectKeyFromObject(f.dtp), &dtprometheus.DTPrometheus{}) == nil
	}, 10*time.Second, 50*time.Millisecond, "the RoleBinding never took effect")
}

// chartRoleRules extracts the rules of one Role from a helm template under
// config/helm/chart/default/templates/Common. The prometheus templates are plain YAML apart from
// a handful of Go template directives in metadata (the release namespace, the shared label
// include and the feature gate), so dropping every line that contains one leaves a parseable
// document. Any change that makes that untrue fails loudly here rather than silently skipping the
// check.
func chartRoleRules(t *testing.T, relativePath, roleName string) []rbacv1.PolicyRule {
	t.Helper()

	path := filepath.Join(projectpath.Root, "config", "helm", "chart", "default", "templates", "Common", relativePath)

	raw, err := readChartTemplate(path)
	require.NoErrorf(t, err, "cannot read chart template %s", path)

	for document := range strings.SplitSeq(raw, "\n---\n") {
		role := &rbacv1.Role{}
		if err := yaml.Unmarshal([]byte(document), role); err != nil {
			continue
		}

		if role.Kind == "Role" && role.Name == roleName {
			require.NotEmptyf(t, role.Rules, "Role %s in %s has no rules, the template is probably no longer parseable", roleName, relativePath)

			return role.Rules
		}
	}

	require.Failf(t, "role not found", "no Role named %s in %s", roleName, relativePath)

	return nil
}

func readChartTemplate(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.Contains(line, "{{") {
			continue
		}

		kept = append(kept, line)
	}

	return strings.Join(kept, "\n"), nil
}

func grantInNamespace(t *testing.T, clt client.Client, namespace, serviceAccountName string, rules []rbacv1.PolicyRule) {
	t.Helper()

	integrationtests.CreateKubernetesObject(t, clt, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: namespace},
	})

	integrationtests.CreateKubernetesObject(t, clt, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: namespace},
		Rules:      rules,
	})

	integrationtests.CreateKubernetesObject(t, clt, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: namespace},
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: serviceAccountName},
		Subjects: []rbacv1.Subject{
			{Kind: rbacv1.ServiceAccountKind, Name: serviceAccountName, Namespace: namespace},
		},
	})
}

func impersonatingClient(t *testing.T, cfg *rest.Config, namespace, serviceAccountName string) client.Client {
	t.Helper()

	impersonated := rest.CopyConfig(cfg)
	impersonated.Impersonate = rest.ImpersonationConfig{
		UserName: "system:serviceaccount:" + namespace + ":" + serviceAccountName,
	}

	clt, err := client.New(impersonated, client.Options{})
	require.NoError(t, err)

	return clt
}
