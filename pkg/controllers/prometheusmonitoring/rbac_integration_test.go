// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/projectpath"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
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
// Only the namespaced Roles are modeled. The PrometheusMonitoring reconcile is namespace scoped: it
// reads a PrometheusMonitoring and a DynaKube, reads the token Secret, and writes ConfigMaps,
// Services, a StatefulSet and two Deployments, all in the PrometheusMonitoring namespace.
func testOperatorRBAC(t *testing.T, clt client.Client, cfg *rest.Config) {
	const serviceAccountName = "dynatrace-operator"

	f := newFixture(t, clt, "rbac")

	rules := chartRole(t, "operator/role-operator.yaml", "dynatrace-operator").Rules
	rules = append(rules,
		chartRole(t, "prometheus/role-prometheusmonitoring-operator.yaml", "dynatrace-prometheusmonitoring-operator").Rules...)
	require.NotEmpty(t, rules)

	grantInNamespace(t, clt, f.ns, serviceAccountName, rules)

	impersonating := impersonatingClient(t, cfg, f.ns, serviceAccountName)
	waitForGrant(t, impersonating, f)

	r := newStubbedReconciler(t, impersonating)

	_, err := r.Reconcile(t.Context(), f.request())
	require.NoError(t, err, "the helm chart does not grant the operator everything the reconcile needs")

	// A reconcile that silently skipped work would also produce no error, so confirm the
	// impersonated run actually created everything.
	assert.ElementsMatch(t, expectedManagedObjects(f.pm), f.listManagedObjects(t))

	t.Run("a second reconcile stays permitted", func(t *testing.T) {
		// Updates need different verbs than creates, and the first reconcile only exercised creates.
		// One field per component, picked so that the StatefulSet, both Deployments and a ConfigMap
		// are all rewritten: changing only the gateway replicas would leave a chart that forgot
		// `update` on deployments or configmaps passing.
		beforeHashes := f.configHashes(t)

		f.pm.Spec.Gateway.Replicas = new(int32(2))
		f.pm.Spec.Scraper.Replicas = new(int32(3))
		f.pm.Spec.TargetAllocator.ScrapeInterval = new(metav1.Duration{Duration: 30 * time.Second})
		f.updatePrometheusMonitoring(t)

		_, err := r.Reconcile(t.Context(), f.request())
		require.NoError(t, err)

		// The StatefulSet and both Deployments.
		assert.Equal(t, new(int32(2)), f.gatewayStatefulSet(t).Spec.Replicas)
		assert.Equal(t, new(int32(3)), f.deployment(t, f.pm.Scraper().GetDeploymentName()).Spec.Replicas)
		assert.NotEqual(t, beforeHashes["targetallocator"], f.configHashes(t)["targetallocator"],
			"the target allocator Deployment was not rewritten")

		// ... and the ConfigMap the new scrape interval is rendered into.
		assert.Equal(t, "30s", f.targetAllocatorConfig(t).PrometheusCR.ScrapeInterval)
	})

	t.Run("the informer cache permissions are granted", func(t *testing.T) {
		// A reconcile only needs get/create/update, but the controller never runs on bare API calls:
		// everything SetupWithManager registers is served from an informer cache, which needs list
		// and watch on every one of those kinds before the manager can even start. A missing
		// list/watch is therefore a crash loop on a real cluster that no reconcile test can see.
		for _, gvk := range watchedKinds(t) {
			resource := resourceFor(t, clt, gvk)

			t.Run(resource.String(), func(t *testing.T) {
				assertAccessAllowed(t, impersonating, f.ns, resource, "list", "watch")
			})
		}
	})

	t.Run("the grant is tight enough to be meaningful", func(t *testing.T) {
		// Guards against the test passing because impersonation is not enforced at all. Creating a
		// Namespace would be the obvious probe, but it proves nothing: a namespaced Role can never
		// grant a cluster scoped resource, so that check would pass even against a Role granting *
		// on everything namespaced. Pods are namespaced and the chart deliberately grants them
		// read-only, so creating one is both within the Role's reach and outside its rules.
		assertRulesDoNotGrant(t, rules, "", "pods", "create")

		err := impersonating.Create(t.Context(), &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "rbac-should-not-be-creatable", Namespace: f.ns},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "test", Image: "registry.example.com/test:1.0.0"}},
			},
		})
		require.Error(t, err)
		assert.True(t, k8serrors.IsForbidden(err), "expected Forbidden, got %v", err)
	})
}

// testComponentRBAC covers the RBAC of the deployed components rather than the operator's, which is
// a different failure mode with the same blast radius: the Forbidden on podmonitors behind ICP-8196
// came from the target allocator's own ServiceAccount. The components are pods, so nothing they do
// passes through the operator's client and no reconcile test can observe it. What is checkable
// without a cluster is whether the chart's per-component ClusterRole covers what the config the
// operator just rendered asks that component to do.
func testComponentRBAC(t *testing.T, clt client.Client, cfg *rest.Config) {
	f := newFixture(t, clt, "component-rbac")
	f.reconcileSuccessfully(t)

	for _, c := range components() {
		t.Run(c.name, func(t *testing.T) {
			tmpl, _ := c.podSpec(t, f)
			serviceAccountName := tmpl.Spec.ServiceAccountName
			require.NotEmpty(t, serviceAccountName)

			clusterRole := chartClusterRole(t, c.clusterRoleTemplate)
			binding := chartClusterRoleBinding(t, c.clusterRoleBindingTemplate)

			// The chart's own wiring rather than an assumption about it: a ClusterRoleBinding that
			// points at the wrong ClusterRole or ServiceAccount leaves the component with no
			// permissions at all, which looks exactly like a missing rule from inside the pod.
			assert.Equal(t, "ClusterRole", binding.RoleRef.Kind)
			assert.Equal(t, clusterRole.Name, binding.RoleRef.Name)
			require.Len(t, binding.Subjects, 1)
			assert.Equal(t, rbacv1.ServiceAccountKind, binding.Subjects[0].Kind)
			assert.Equal(t, serviceAccountName, binding.Subjects[0].Name,
				"the chart binds a different ServiceAccount than the one the pod template runs as")

			grantClusterWide(t, clt, f.ns, serviceAccountName, clusterRole.Rules)

			impersonating := impersonatingClient(t, cfg, f.ns, serviceAccountName)

			// Cluster wide: the allocator's namespace selectors and the collector's k8s processors
			// are not restricted to the release namespace.
			for _, access := range requiredComponentAccess(t, f, c.name) {
				assertAccessAllowed(t, impersonating, "", access, "list", "watch")
			}
		})
	}

	t.Run("every ServiceAccount the pods run as exists in the chart", func(t *testing.T) {
		// The operator only references the ServiceAccounts by name, so a chart that renames or drops
		// one produces pods that never start, with nothing in this repo to catch it.
		inChart := chartServiceAccountNames(t, "prometheus")
		require.NotEmpty(t, inChart)

		for _, c := range components() {
			tmpl, _ := c.podSpec(t, f)
			assert.Containsf(t, inChart, tmpl.Spec.ServiceAccountName,
				"the %s pod template runs as a ServiceAccount the chart does not create (chart has %v)",
				c.name, inChart)
		}
	})
}

// requiredComponentAccess derives which API resources a component's ServiceAccount has to be able
// to read from the config the operator just rendered for it, rather than from a wish list that can
// drift away from what the component is configured to do. The step from a config feature to the
// resources that feature reads is the one part written down here, because that knowledge lives in
// the upstream collector and allocator, not in this repository.
func requiredComponentAccess(t *testing.T, f *fixture, componentName string) []apiResource {
	t.Helper()

	switch componentName {
	case "gateway":
		return collectorAccess(t, f.gatewayConfig(t))
	case "scraper":
		return collectorAccess(t, f.scraperConfig(t))
	default:
		return targetAllocatorAccess(t, f)
	}
}

// collectorAccess maps the Kubernetes-aware pieces of an OTel Collector config to what they read.
func collectorAccess(t *testing.T, cfg collectorConfig) []apiResource {
	t.Helper()

	access := make([]apiResource, 0)

	// The k8s_attributes processor enriches every datapoint from the pod it came from, that pod's
	// namespace and its owning workload, all of which it watches cluster wide.
	if _, ok := cfg.Processors["k8s_attributes"]; ok {
		access = append(access,
			apiResource{"", "pods"},
			apiResource{"", "namespaces"},
			apiResource{"apps", "replicasets"},
		)
	}

	// The load balancing exporter's k8s resolver keeps the gateway pool membership current from the
	// endpoints behind the gateway Service.
	if raw, ok := cfg.Exporters["load_balancing"]; ok {
		exporter := unmarshalInto[loadBalancingExporter](t, raw, "load_balancing exporter")
		require.NotEmpty(t, exporter.Resolver.K8s.Service,
			"the k8s resolver is the only load balancing resolver the chart grants for")

		access = append(access,
			apiResource{"", "services"},
			apiResource{"", "endpoints"},
			apiResource{"discovery.k8s.io", "endpointslices"},
		)
	}

	require.NotEmpty(t, access, "the rendered config reads nothing from the API server, is that still true?")

	return access
}

// targetAllocatorAccess derives the allocator's access from its rendered config: every Prometheus
// CRD kind it is configured to select has to be readable, and the Prometheus service discovery it
// runs on top of those CRs reads the objects they point at.
func targetAllocatorAccess(t *testing.T, f *fixture) []apiResource {
	t.Helper()

	cfg := f.targetAllocatorConfig(t)
	require.True(t, cfg.PrometheusCR.Enabled, "the allocator only reads the API server in prometheus_cr mode")
	require.NotEmpty(t, cfg.CollectorNamespace)
	require.NotNil(t, cfg.CollectorSelector)

	access := []apiResource{
		// Service discovery for the selected monitors, plus the scraper pods (collector_selector)
		// the discovered targets are handed out to.
		{"", "namespaces"},
		{"", "pods"},
		{"", "services"},
		{"discovery.k8s.io", "endpointslices"},
	}

	for _, resource := range prometheusCRResources(t, f) {
		access = append(access, apiResource{"monitoring.coreos.com", resource})
	}

	return access
}

// prometheusCRResources reads the Prometheus CRD kinds the allocator is configured to select out of
// the rendered prometheus_cr block, so a fifth kind added to the config without a matching chart
// rule fails here rather than on a customer cluster.
func prometheusCRResources(t *testing.T, f *fixture) []string {
	t.Helper()

	byConfigKey := map[string]string{
		"pod_monitor":     "podmonitors",
		"service_monitor": "servicemonitors",
		"scrape_config":   "scrapeconfigs",
		"probe":           "probes",
	}

	resources := make([]string, 0, len(byConfigKey))

	for key := range f.targetAllocatorPrometheusCR(t) {
		prefix, isSelector := strings.CutSuffix(key, "_selector")
		if !isSelector || strings.HasSuffix(prefix, "_namespace") {
			continue
		}

		resource, known := byConfigKey[prefix]
		require.Truef(t, known, "%s has no RBAC mapping, does the chart grant that CRD?", key)

		resources = append(resources, resource)
	}

	require.Len(t, resources, len(byConfigKey), "the allocator must select every Prometheus CRD kind")

	return resources
}

// apiResource is one group/resource pair to run an access review against.
type apiResource struct {
	group    string
	resource string
}

func (r apiResource) String() string {
	if r.group == "" {
		return r.resource
	}

	return r.resource + "." + r.group
}

const (
	grantTimeout  = 10 * time.Second
	grantInterval = 50 * time.Millisecond
)

// assertAccessAllowed asks the API server, as the impersonated ServiceAccount, whether verb is
// permitted on the resource. A SelfSubjectAccessReview is the only way to check list and watch
// without opening one, and unlike a List call it answers for the exact group/resource/namespace
// triple an informer would use. An empty namespace asks the cluster wide question.
//
// It polls because the grant is written through the API server but reaches the authorizer through
// an informer, so the first review after a fresh binding can still come back denied.
func assertAccessAllowed(t *testing.T, impersonating client.Client, namespace string, resource apiResource, verbs ...string) {
	t.Helper()

	for _, verb := range verbs {
		deadline := time.Now().Add(grantTimeout)

		for {
			review := &authorizationv1.SelfSubjectAccessReview{
				Spec: authorizationv1.SelfSubjectAccessReviewSpec{
					ResourceAttributes: &authorizationv1.ResourceAttributes{
						Namespace: namespace,
						Group:     resource.group,
						Resource:  resource.resource,
						Verb:      verb,
					},
				},
			}
			require.NoError(t, impersonating.Create(t.Context(), review))

			if review.Status.Allowed {
				break
			}

			if time.Now().After(deadline) {
				require.Failf(t, "access not granted",
					"the chart does not grant %s on %s: %s", verb, resource, review.Status.Reason)
			}

			time.Sleep(grantInterval)
		}
	}
}

// builderCallPattern matches the object argument of a controller builder call, for example
// `Owns(&appsv1.Deployment{})` or `Watches(&dynakube.DynaKube{},`, capturing the package alias and
// the type name. Whitespace is allowed on both sides of the call because the builder calls are
// chained one per line (the dot ends the previous line) and Watches spreads its arguments over
// several lines.
var builderCallPattern = regexp.MustCompile(`\.\s*(?:For|Owns|Watches)\(\s*&(\w+)\.(\w+)\{\}`)

// importPattern matches one line of an import block, capturing the optional alias and the path.
var importPattern = regexp.MustCompile(`^\s*(?:(\w+) )?"([^"]+)"$`)

// watchedKinds returns the object kinds SetupWithManager registers with the manager, read out of
// the source rather than restated here: the For, Owns and Watches calls in reconciler.go. An Owns()
// added later without a matching chart rule therefore fails the informer cache check above.
//
// The source is scanned as text rather than parsed, because the go/ast packages are not on this
// repository's allowed import list. That is good enough here: the call sites are one-liners in a
// known file, and the count check below turns a pattern that stops matching into a failure instead
// of an empty check.
func watchedKinds(t *testing.T) []schema.GroupVersionKind {
	t.Helper()

	const sourceFile = "reconciler.go"

	content, err := os.ReadFile(sourceFile)
	require.NoErrorf(t, err, "cannot read %s", sourceFile)

	source := string(content)
	imports := sourceImports(source)
	kinds := make([]schema.GroupVersionKind, 0)

	for _, match := range builderCallPattern.FindAllStringSubmatch(source, -1) {
		alias, typeName := match[1], match[2]

		importPath, known := imports[alias]
		require.Truef(t, known, "no import found for package %s in %s", alias, sourceFile)

		kinds = append(kinds, kindOfType(t, importPath, typeName))
	}

	// For plus four Owns plus one Watches today. Any drop below that means the scan stopped
	// recognizing the builder calls and the check silently became empty.
	require.GreaterOrEqual(t, len(kinds), 6, "SetupWithManager registers fewer kinds than expected")

	return kinds
}

// sourceImports maps the package alias each import is referred to by, which is its explicit alias
// or the last segment of its path, to the import path itself.
func sourceImports(source string) map[string]string {
	imports := make(map[string]string)

	for line := range strings.Lines(source) {
		match := importPattern.FindStringSubmatch(strings.TrimSuffix(line, "\n"))
		if match == nil {
			continue
		}

		alias, path := match[1], match[2]
		if alias == "" {
			alias = path[strings.LastIndex(path, "/")+1:]
		}

		imports[alias] = path
	}

	return imports
}

// kindOfType looks the Go type up in the scheme the manager uses, so the group and version come
// from the type's own registration instead of a second list to keep in sync.
func kindOfType(t *testing.T, importPath, typeName string) schema.GroupVersionKind {
	t.Helper()

	found := make([]schema.GroupVersionKind, 0, 1)

	for gvk, goType := range scheme.Scheme.AllKnownTypes() {
		if goType.PkgPath() == importPath && goType.Name() == typeName {
			found = append(found, gvk)
		}
	}

	require.Lenf(t, found, 1, "%s.%s is registered under %v, cannot pick one", importPath, typeName, found)

	return found[0]
}

// resourceFor turns a kind into the group/resource pair RBAC is written against, through the
// discovery information of the running API server.
func resourceFor(t *testing.T, clt client.Client, gvk schema.GroupVersionKind) apiResource {
	t.Helper()

	mapping, err := clt.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	require.NoErrorf(t, err, "cannot map %s to a resource", gvk)

	return apiResource{group: mapping.Resource.Group, resource: mapping.Resource.Resource}
}

// assertRulesDoNotGrant fails when rules permit verb on group/resource, wildcards included. The
// Forbidden assertion it guards is only meaningful while that holds, so this keeps the negative
// test self-validating: widening the chart rules fails here instead of quietly turning the
// Forbidden check into a tautology.
func assertRulesDoNotGrant(t *testing.T, rules []rbacv1.PolicyRule, group, resource, verb string) {
	t.Helper()

	for _, rule := range rules {
		if ruleCovers(rule.APIGroups, group) && ruleCovers(rule.Resources, resource) && ruleCovers(rule.Verbs, verb) {
			require.Failf(t, "the negative check is no longer negative",
				"the chart grants %s on %s (apiGroup %q), pick something it does not grant", verb, resource, group)
		}
	}
}

func ruleCovers(values []string, want string) bool {
	return slices.Contains(values, rbacv1.ResourceAll) || slices.Contains(values, want)
}

// waitForGrant blocks until the API server's authorizer has picked up the freshly created
// RoleBinding. The binding is written through the same API server, but the authorizer sees it via
// an informer, so a reconcile started immediately can still be rejected.
func waitForGrant(t *testing.T, impersonating client.Client, f *fixture) {
	t.Helper()

	require.Eventually(t, func() bool {
		return impersonating.Get(t.Context(), client.ObjectKeyFromObject(f.pm), &prometheusmonitoring.PrometheusMonitoring{}) == nil
	}, grantTimeout, grantInterval, "the RoleBinding never took effect")
}

// chartTemplateDir is config/helm/chart/default/templates/Common, which holds the operator's own
// RBAC and the prometheus subdirectory with everything PrometheusMonitoring adds.
func chartTemplateDir() string {
	return filepath.Join(projectpath.Root, "config", "helm", "chart", "default", "templates", "Common")
}

// chartDocument is one YAML document from a helm template, with the Go template directives removed.
type chartDocument struct {
	yaml string

	// conditionalRules is true when at least one line after the document's rules: key was dropped
	// because it sat behind a Go template conditional. Such a rule only exists for some
	// installations, so counting it as granted would over-state the grant.
	conditionalRules bool
}

// chartDocuments reads a helm template and strips its Go template directives so the remainder
// parses as YAML. The prometheus and operator RBAC templates are plain YAML apart from a handful of
// directives in metadata (the release namespace, the shared label include, the feature gate) and
// the occasional OpenShift-only rule.
//
// Lines inside a conditional or range block that starts after a rules: key are dropped along with
// the directive: keeping them would count a rule that only exists on OpenShift as granted
// everywhere, which is the direction that makes an RBAC test pass while a real cluster fails. The
// drop is recorded so callers that need the complete rule set can insist there was nothing to drop.
func chartDocuments(t *testing.T, relativePath string) []chartDocument {
	t.Helper()

	path := filepath.Join(chartTemplateDir(), relativePath)

	content, err := os.ReadFile(path)
	require.NoErrorf(t, err, "cannot read chart template %s", path)

	documents := make([]chartDocument, 0, 1)
	current := chartDocument{}
	kept := make([]string, 0)
	inRules := false
	depth := 0

	flush := func() {
		current.yaml = strings.Join(kept, "\n")
		documents = append(documents, current)
		current = chartDocument{}
		kept = make([]string, 0)
		inRules = false
		depth = 0
	}

	for line := range strings.Lines(string(content)) {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\n"))

		if trimmed == "---" {
			flush()

			continue
		}

		switch {
		case !strings.Contains(trimmed, "{{"):
			if depth == 0 {
				kept = append(kept, strings.TrimSuffix(line, "\n"))
			}

			if trimmed == "rules:" {
				inRules = true
			}
		case inRules && isBlockStart(trimmed):
			depth++
			current.conditionalRules = true
		case inRules && isBlockEnd(trimmed) && depth > 0:
			depth--
		}
	}

	flush()

	return documents
}

func isBlockStart(line string) bool {
	for _, keyword := range []string{"if", "range", "with"} {
		if strings.HasPrefix(line, "{{ "+keyword+" ") || strings.HasPrefix(line, "{{- "+keyword+" ") {
			return true
		}
	}

	return false
}

func isBlockEnd(line string) bool {
	return strings.HasPrefix(line, "{{ end") || strings.HasPrefix(line, "{{- end")
}

// chartRole extracts one namespaced Role from a helm template. Its rules are used verbatim as the
// grant the reconcile then runs under, so a conditional rule inside it is refused rather than
// silently dropped: the test would otherwise decide for itself which installation it is checking.
func chartRole(t *testing.T, relativePath, name string) rbacv1.Role {
	t.Helper()

	for _, document := range chartDocuments(t, relativePath) {
		role := rbacv1.Role{}
		if yaml.Unmarshal([]byte(document.yaml), &role) != nil || role.Kind != "Role" || role.Name != name {
			continue
		}

		require.Falsef(t, document.conditionalRules,
			"Role %s in %s has rules behind a template conditional, so its full rule set depends on the "+
				"helm values and can no longer be modeled here", name, relativePath)
		require.NotEmptyf(t, role.Rules,
			"Role %s in %s has no rules, the template is probably no longer parseable", name, relativePath)

		return role
	}

	require.Failf(t, "role not found", "no Role named %s in %s", name, relativePath)

	return rbacv1.Role{}
}

// chartClusterRole extracts the single ClusterRole from a helm template. Unlike chartRole it
// tolerates conditional rules: those are dropped, and since every assertion against a ClusterRole
// is "this must be granted", dropping a rule can only make the test stricter.
func chartClusterRole(t *testing.T, relativePath string) rbacv1.ClusterRole {
	t.Helper()

	found := make([]rbacv1.ClusterRole, 0, 1)

	for _, document := range chartDocuments(t, relativePath) {
		clusterRole := rbacv1.ClusterRole{}
		if yaml.Unmarshal([]byte(document.yaml), &clusterRole) != nil || clusterRole.Kind != "ClusterRole" {
			continue
		}

		require.NotEmptyf(t, clusterRole.Rules,
			"ClusterRole %s in %s has no unconditional rules", clusterRole.Name, relativePath)

		found = append(found, clusterRole)
	}

	require.Lenf(t, found, 1, "expected exactly one ClusterRole in %s", relativePath)

	return found[0]
}

func chartClusterRoleBinding(t *testing.T, relativePath string) rbacv1.ClusterRoleBinding {
	t.Helper()

	found := make([]rbacv1.ClusterRoleBinding, 0, 1)

	for _, document := range chartDocuments(t, relativePath) {
		binding := rbacv1.ClusterRoleBinding{}
		if yaml.Unmarshal([]byte(document.yaml), &binding) != nil || binding.Kind != "ClusterRoleBinding" {
			continue
		}

		found = append(found, binding)
	}

	require.Lenf(t, found, 1, "expected exactly one ClusterRoleBinding in %s", relativePath)

	return found[0]
}

// chartServiceAccountNames returns the names of every ServiceAccount the templates in one chart
// subdirectory create.
func chartServiceAccountNames(t *testing.T, relativeDir string) []string {
	t.Helper()

	pattern := filepath.Join(chartTemplateDir(), relativeDir, "serviceaccount-*.yaml")

	paths, err := filepath.Glob(pattern)
	require.NoErrorf(t, err, "cannot list %s", pattern)
	require.NotEmptyf(t, paths, "no ServiceAccount templates match %s", pattern)

	names := make([]string, 0, len(paths))

	for _, path := range paths {
		relative, err := filepath.Rel(chartTemplateDir(), path)
		require.NoError(t, err)

		for _, document := range chartDocuments(t, relative) {
			serviceAccount := corev1.ServiceAccount{}
			if yaml.Unmarshal([]byte(document.yaml), &serviceAccount) != nil || serviceAccount.Kind != "ServiceAccount" {
				continue
			}

			names = append(names, serviceAccount.Name)
		}
	}

	return names
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

// grantClusterWide models what the chart does for a component: a ServiceAccount in the release
// namespace, a ClusterRole, and a ClusterRoleBinding between them. The cluster scoped names are
// derived from the fixture namespace so repeated runs and the other components do not collide;
// CreateKubernetesObject removes them again when the test ends.
func grantClusterWide(t *testing.T, clt client.Client, namespace, serviceAccountName string, rules []rbacv1.PolicyRule) {
	t.Helper()

	require.NotEmpty(t, rules)

	name := namespace + "-" + serviceAccountName

	integrationtests.CreateKubernetesObject(t, clt, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: namespace},
	})

	integrationtests.CreateKubernetesObject(t, clt, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Rules:      rules,
	})

	integrationtests.CreateKubernetesObject(t, clt, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: name},
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
