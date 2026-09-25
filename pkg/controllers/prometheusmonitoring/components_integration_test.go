// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the PrometheusMonitoring controller as a whole, against a real API server.
//
// The per-component reconcilers already have their own envtest lifecycle tests
// (gateway/scraper/targetallocator ...reconciler_integration_test.go) plus golden-file unit
// tests for the exact rendered shape. What is only observable here is the result of one full
// Reconcile: the complete set of objects the three component reconcilers create together, the
// references between them (Service <-> container port, config <-> Service DNS name, ConfigMap and
// Secret references that must resolve), and the status the top-level reconciler derives from them.
//
// envtest has no kube-controller-manager and no kubelet, so there are no pods, no workload status
// and no garbage collection. Ownership is therefore asserted through owner references rather than
// through cascading deletion, and readiness is out of scope (it belongs to the e2e follow-up).

const (
	// Distinctive so the leak check can search for them verbatim.
	testAPIToken        = "dt0c01.INTEGRATIONAPITOKENVALUE"
	testDataIngestToken = "dt0c01.INTEGRATIONDATAINGESTTOKENVALUE"

	// The images the stubbed fleet management API hands out. Gateway and scraper are both plain
	// OTel Collectors and therefore share one image, exactly as they do in production.
	testCollectorImage       = "registry.example.com/dynatrace/otel-collector:1.2.3"
	testTargetAllocatorImage = "registry.example.com/dynatrace/target-allocator:1.2.3"

	testDynaKubeName = "dk"
	testAPIURL       = "https://tenant.dev.dynatracelabs.com/api"
	testClusterName  = "integration-cluster"

	// How long to wait on a live Manager to notice something and act on it.
	managerTimeout  = 30 * time.Second
	managerInterval = 100 * time.Millisecond
)

// component describes one deployed PrometheusMonitoring component so that the aspect tests below can be
// table driven. Adding a component later (for example Self Monitoring, which is intentionally not
// covered yet) means adding one entry here plus its expected object set.
type component struct {
	name string
	// objectName is the Deployment or StatefulSet, and also the ConfigMap and Service name:
	// all objects of a component share one name.
	objectName func(pm *prometheusmonitoring.PrometheusMonitoring) string
	labels     func() *k8slabel.Labels
	// getWorkload returns the component's Deployment or StatefulSet.
	getWorkload   func(t *testing.T, f *fixture) workload
	containerName string
	// hasService is false for the scraper: nothing connects to it inbound.
	hasService bool
	// serviceAccount is created by the helm chart, not by the operator.
	serviceAccount string
	// expectedImage is the image the reconciler has to resolve from the fleet management API,
	// since a default PrometheusMonitoring carries no image in its spec.
	expectedImage string
	// expectedUser is the UID the container has to run as, spelled out here rather than taken from
	// the production constant so that changing the constant fails this test.
	expectedUser int64
}

// workload is a Deployment or StatefulSet reduced to what the tests need: the object itself, so a
// test can write it back, plus the pod template and selector both kinds carry.
type workload struct {
	object   client.Object
	template *corev1.PodTemplateSpec
	selector *metav1.LabelSelector
}

func components() []component {
	return []component{
		{
			name:           "gateway",
			objectName:     func(pm *prometheusmonitoring.PrometheusMonitoring) string { return pm.Gateway().GetStatefulSetName() },
			labels:         func() *k8slabel.Labels { return k8slabel.New("opentelemetry-gateway", "otel-gateway", "") },
			getWorkload:    gatewayWorkload,
			containerName:  "gateway",
			hasService:     true,
			serviceAccount: "dynatrace-prometheus-gateway",
			expectedImage:  testCollectorImage,
			expectedUser:   10001,
		},
		{
			name:           "scraper",
			objectName:     func(pm *prometheusmonitoring.PrometheusMonitoring) string { return pm.Scraper().GetDeploymentName() },
			labels:         k8slabel.OTelScraper,
			getWorkload:    scraperWorkload,
			containerName:  "scraper",
			hasService:     false,
			serviceAccount: "dynatrace-prometheus-scraper",
			expectedImage:  testCollectorImage,
			expectedUser:   65532,
		},
		{
			name: "targetallocator",
			objectName: func(pm *prometheusmonitoring.PrometheusMonitoring) string {
				return pm.TargetAllocator().GetDeploymentName()
			},
			labels:         k8slabel.OTelTargetAllocator,
			getWorkload:    targetAllocatorWorkload,
			containerName:  "targetallocator",
			hasService:     true,
			serviceAccount: "dynatrace-target-allocator",
			expectedImage:  testTargetAllocatorImage,
			expectedUser:   65532,
		},
	}
}

func gatewayWorkload(t *testing.T, f *fixture) workload {
	t.Helper()

	sts := f.getGatewayStatefulSet(t)

	return workload{object: sts, template: &sts.Spec.Template, selector: sts.Spec.Selector}
}

func scraperWorkload(t *testing.T, f *fixture) workload {
	t.Helper()

	deploy := f.getDeployment(t, f.pm.Scraper().GetDeploymentName())

	return workload{object: deploy, template: &deploy.Spec.Template, selector: deploy.Spec.Selector}
}

func targetAllocatorWorkload(t *testing.T, f *fixture) workload {
	t.Helper()

	deploy := f.getDeployment(t, f.pm.TargetAllocator().GetDeploymentName())

	return workload{object: deploy, template: &deploy.Spec.Template, selector: deploy.Spec.Selector}
}

// TestReconcileComponents drives the full top-level Reconcile against a real API server.
// One envtest control plane is shared by all subtests; each subtest gets its own namespace so the
// object sets never overlap.
func TestReconcileComponents(t *testing.T) {
	clt, cfg := integrationtests.SetupTestEnvironmentWithConfig(t)

	t.Run("creation", func(t *testing.T) { testCreation(t, clt) })
	t.Run("workload-spec", func(t *testing.T) { testWorkloadSpec(t, clt) })
	t.Run("references-resolve", func(t *testing.T) { testReferencesResolve(t, clt) })
	t.Run("wiring", func(t *testing.T) { testWiring(t, clt) })
	t.Run("configmaps", func(t *testing.T) { testConfigMaps(t, clt) })
	t.Run("secrets", func(t *testing.T) { testSecrets(t, clt) })
	t.Run("update-strategies", func(t *testing.T) { testUpdateStrategies(t, clt) })
	t.Run("idempotency", func(t *testing.T) { testIdempotency(t, clt) })
	t.Run("update-propagation", func(t *testing.T) { testUpdatePropagation(t, clt) })
	t.Run("drift-correction", func(t *testing.T) { testDriftCorrection(t, clt) })
	t.Run("status", func(t *testing.T) { testStatus(t, clt) })
	t.Run("error-paths", func(t *testing.T) { testErrorPaths(t, clt) })
	t.Run("drift-correction-under-manager", func(t *testing.T) { testDriftCorrectionUnderManager(t, clt, cfg) })
}

// managedObject identifies one object the controller is expected to manage.
type managedObject struct {
	kind string
	name string
}

func (o managedObject) String() string {
	return o.kind + "/" + o.name
}

// expectedManagedObjects is the complete inventory of what one PrometheusMonitoring reconcile creates.
// The creation test asserts equality against it, so a component that starts creating an extra
// object (or stops creating one) fails here rather than silently drifting.
func expectedManagedObjects(pm *prometheusmonitoring.PrometheusMonitoring) []managedObject {
	gw := pm.Gateway().GetStatefulSetName()
	sc := pm.Scraper().GetDeploymentName()
	ta := pm.TargetAllocator().GetDeploymentName()

	return []managedObject{
		{"ConfigMap", gw},
		{"StatefulSet", gw},
		{"Service", gw},
		{"ConfigMap", sc},
		{"Deployment", sc},
		{"ConfigMap", ta},
		{"Deployment", ta},
		{"Service", ta},
	}
}

func testCreation(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "creation")
	f.assertReconcileSuccessfully(t)

	assert.ElementsMatch(t, expectedManagedObjects(f.pm), f.listManagedObjects(t),
		"the set of objects created by a full reconcile changed")

	t.Run("nothing else is created", func(t *testing.T) {
		// The inventory above filters on the managed-by label, so an object created without that
		// label is invisible to it. Listing the same kinds unfiltered closes that hole, and subsumes
		// two narrower claims at once: that the controller creates no Secret of its own (the DynaKube
		// token Secret is only ever read and referenced, never copied) and no ServiceAccount (the
		// components' ServiceAccounts come from the helm chart, the operator only names them).
		want := append(expectedManagedObjects(f.pm), fixtureObjects()...)

		assert.ElementsMatch(t, want, f.listAllObjects(t),
			"an object exists in the namespace that is neither part of the fixture nor an expected managed object")
	})

	t.Run("every managed object is owned by the PrometheusMonitoring", func(t *testing.T) {
		// The exact owner reference (apiVersion, kind, blockOwnerDeletion) is pinned per object by
		// the component golden files. What only the full inventory can show is that nothing is
		// created unowned, which would survive the cascade and leak on delete.
		for _, want := range expectedManagedObjects(f.pm) {
			assert.Truef(t, metav1.IsControlledBy(f.getManagedObject(t, want), f.pm),
				"%s is not controlled by the PrometheusMonitoring", want)
		}
	})

	t.Run("no finalizer is added", func(t *testing.T) {
		// Cleanup relies entirely on the owner references above. A finalizer nobody removes would
		// wedge deletion, and there is no code to remove one.
		assert.Empty(t, f.getStoredPrometheusMonitoring(t).Finalizers)
	})
}

func testWorkloadSpec(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "workload-spec")
	f.assertReconcileSuccessfully(t)

	t.Run("workload spec is correct", func(t *testing.T) {
		for _, c := range components() {
			t.Run(c.name, func(t *testing.T) { assertWorkloadSpec(t, f, c) })
		}
	})

	t.Run("service target ports resolve to a container port", func(t *testing.T) {
		for _, c := range components() {
			if !c.hasService {
				continue
			}

			t.Run(c.name, func(t *testing.T) { assertServiceRoutesToWorkload(t, f, c) })
		}
	})
}

func assertWorkloadSpec(t *testing.T, f *fixture, c component) {
	t.Helper()

	w := c.getWorkload(t, f)
	container := containerByName(t, w.template, c.containerName)

	t.Run("selector matches the pod template labels", func(t *testing.T) {
		require.NotNil(t, w.selector)
		assert.NotEmpty(t, w.selector.MatchLabels)

		for key, value := range w.selector.MatchLabels {
			assert.Equal(t, value, w.template.Labels[key], "pod template is missing selector label %s", key)
		}

		assert.Equal(t, c.labels().AsSelector(), w.selector.MatchLabels)
	})

	t.Run("service account and token automounting", func(t *testing.T) {
		assert.Equal(t, c.serviceAccount, w.template.Spec.ServiceAccountName)
		assert.Equal(t, new(true), w.template.Spec.AutomountServiceAccountToken)
	})

	t.Run("container hardening", func(t *testing.T) {
		sec := container.SecurityContext
		require.NotNil(t, sec)

		assert.Equal(t, new(false), sec.Privileged)
		assert.Equal(t, new(false), sec.AllowPrivilegeEscalation)
		assert.Equal(t, new(true), sec.RunAsNonRoot)
		assert.Equal(t, new(true), sec.ReadOnlyRootFilesystem)
		assert.Equal(t, new(c.expectedUser), sec.RunAsUser)
		require.NotNil(t, sec.SeccompProfile)
		assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, sec.SeccompProfile.Type)
		require.NotNil(t, sec.Capabilities)
		assert.Equal(t, []corev1.Capability{"ALL"}, sec.Capabilities.Drop)
	})

	// Probes are deliberately not asserted here: each component's golden workload pins both of them
	// in full, and "a probe exists" adds nothing on top of that.

	t.Run("image is the one resolved from the tenant", func(t *testing.T) {
		assert.Equal(t, c.expectedImage, container.Image)
	})
}

func assertServiceRoutesToWorkload(t *testing.T, f *fixture, c component) {
	t.Helper()

	w := c.getWorkload(t, f)
	container := containerByName(t, w.template, c.containerName)
	svc := f.getService(t, c.objectName(f.pm))

	require.NotEmpty(t, svc.Spec.Ports)

	for _, port := range svc.Spec.Ports {
		assertTargetPortExists(t, container, port)
	}

	assert.Equal(t, c.labels().AsSelector(), svc.Spec.Selector,
		"service selector must match the workload pod labels")
}

// assertTargetPortExists checks that a Service port routes to a port the container actually opens.
// All PrometheusMonitoring Services use named target ports, so a rename on either side breaks the Service
// silently on a real cluster; here it fails the test.
func assertTargetPortExists(t *testing.T, container corev1.Container, port corev1.ServicePort) {
	t.Helper()

	names := make([]string, 0, len(container.Ports))
	for _, containerPort := range container.Ports {
		if containerPort.Name == port.TargetPort.StrVal {
			return
		}

		names = append(names, containerPort.Name)
	}

	assert.Failf(t, "unresolvable target port",
		"service port %s targets %q, but the container only exposes %v", port.Name, port.TargetPort.String(), names)
}

// testReferencesResolve checks that every ConfigMap and Secret a pod template points at exists and
// carries the referenced key. envtest never schedules the pods, so a dangling reference would
// otherwise only show up as a stuck pod on a real cluster.
func testReferencesResolve(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "references")
	f.assertReconcileSuccessfully(t)

	for _, c := range components() {
		t.Run(c.name, func(t *testing.T) {
			w := c.getWorkload(t, f)

			for _, volume := range w.template.Spec.Volumes {
				f.assertVolumeSourceResolves(t, volume)
			}

			container := containerByName(t, w.template, c.containerName)
			for _, env := range container.Env {
				f.assertEnvSourceResolves(t, env)
			}

			// A pull secret is referenced by name only, so a missing one is not a config error the
			// operator can see: the pods just sit in ImagePullBackOff.
			for _, pullSecret := range w.template.Spec.ImagePullSecrets {
				f.assertSecretExists(t, pullSecret.Name)
			}

			t.Run("every volume mount has a volume", func(t *testing.T) {
				for _, mount := range container.VolumeMounts {
					assert.True(t, hasVolume(w.template.Spec.Volumes, mount.Name),
						"volumeMount %s has no matching volume", mount.Name)
				}
			})
		})
	}
}

func hasVolume(volumes []corev1.Volume, name string) bool {
	for _, volume := range volumes {
		if volume.Name == name {
			return true
		}
	}

	return false
}

func containerByName(t *testing.T, tmpl *corev1.PodTemplateSpec, name string) corev1.Container {
	t.Helper()

	for _, container := range tmpl.Spec.Containers {
		if container.Name == name {
			return container
		}
	}

	require.Failf(t, "container not found", "no container named %s in the pod template", name)

	return corev1.Container{}
}

// testUpdateStrategies is what is left of the customization coverage. Replicas, labels,
// annotations, pull policy, node selector, tolerations and priority class are all pinned per
// component by the golden workloads, so they are not restated here. Two things are only observable
// from a full reconcile against a real apiserver: that one reconcile routes each of the three
// strategies to the right workload (they are three different fields on two different kinds), and
// the branch of the merge helpers where an explicit non-rolling type has to clear the rollingUpdate
// block the apiserver defaulted in, because the apiserver rejects the two together.
func testUpdateStrategies(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "update-strategies")
	f.assertReconcileSuccessfully(t)

	f.pm.Spec.Gateway.UpdateStrategy = appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType}
	f.pm.Spec.TargetAllocator.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	f.pm.Spec.Scraper.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	f.updatePrometheusMonitoring(t)
	f.assertReconcileSuccessfully(t)

	gatewayStrategy := f.getGatewayStatefulSet(t).Spec.UpdateStrategy
	assert.Equal(t, appsv1.OnDeleteStatefulSetStrategyType, gatewayStrategy.Type)
	assert.Nil(t, gatewayStrategy.RollingUpdate, "OnDelete must not keep a rollingUpdate block")

	for name, deploymentName := range map[string]string{
		"target allocator": f.pm.TargetAllocator().GetDeploymentName(),
		"scraper":          f.pm.Scraper().GetDeploymentName(),
	} {
		strategy := f.getDeployment(t, deploymentName).Spec.Strategy
		assert.Equal(t, appsv1.RecreateDeploymentStrategyType, strategy.Type, name)
		assert.Nil(t, strategy.RollingUpdate, "%s: Recreate must not keep a rollingUpdate block", name)
	}
}

// testIdempotency asserts that repeating a reconcile changes nothing. Each component's own
// lifecycle test already covers that for its own objects; what is only visible from here is the
// top-level reconciler, which writes the PrometheusMonitoring status on every single pass.
func testIdempotency(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "idempotency")
	f.assertReconcileSuccessfully(t)

	before := f.resourceVersions(t)
	beforeOwner := f.getStoredPrometheusMonitoring(t).ResourceVersion

	// resourceVersion alone is not proof: the API server re-defaults an incoming object before
	// comparing it to storage, so a reconcile that sends a stale object still issues a real write
	// that is silently no-op'ed server-side. Counting the calls the reconcile makes catches that,
	// for every kind of write rather than only for Update: a component switching from Update to
	// Patch, or deleting and recreating an object it could have updated, would otherwise slip
	// through while still restarting pods on a real cluster.
	counting := &callCounter{Client: clt}
	f.r = newStubbedReconciler(t, counting)

	for range 3 {
		f.assertReconcileSuccessfully(t)
		assert.Equal(t, before, f.resourceVersions(t))
	}

	assert.Zero(t, counting.creates, "a no-op reconcile must not issue any Create call")
	assert.Zero(t, counting.updates, "a no-op reconcile must not issue any Update call")
	assert.Zero(t, counting.patches, "a no-op reconcile must not issue any Patch call")
	assert.Zero(t, counting.deletes, "a no-op reconcile must not issue any Delete call")

	// The status is applied server-side on every pass (through SubResource("status"), which is why
	// the counter above does not see it). An apply that carries the same content does not bump
	// resourceVersion, so the owner has to stay untouched too; if it does not, the status the
	// reconciler derives is not stable and the controller would write in a loop on a real cluster.
	assert.Equal(t, beforeOwner, f.getStoredPrometheusMonitoring(t).ResourceVersion,
		"a no-op reconcile rewrote the PrometheusMonitoring")
}

// testDriftCorrectionUnderManager runs the real reconciler under a live Manager, which is the only
// way to prove that drift actually reaches it: the drift tests above call Reconcile by hand, so they
// show the correction but not the trigger. It reuses the control plane the other subtests already
// started rather than paying for a second one.
func testDriftCorrectionUnderManager(t *testing.T, clt client.Client, cfg *rest.Config) {
	f := newFixture(t, clt, "drift-manager")

	integrationtests.StartManager(t, cfg, func(mgr ctrl.Manager) error {
		return newStubbedReconciler(t, mgr.GetClient()).SetupWithManager(mgr)
	})

	name := f.pm.Scraper().GetDeploymentName()

	// The Manager reconciles on its own as soon as it sees the PrometheusMonitoring, so wait for
	// that first pass before taking anything away.
	require.Eventually(t, func() bool {
		return f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, &corev1.ConfigMap{}) == nil
	}, managerTimeout, managerInterval, "the initial reconcile never created the scraper ConfigMap")

	deleted := f.getConfigMap(t, name)
	require.NoError(t, clt.Delete(t.Context(), deleted))

	// A new UID rather than just presence: a cached read could otherwise still serve the object
	// that was just deleted and pass without anything having been recreated.
	require.Eventually(t, func() bool {
		restored := &corev1.ConfigMap{}
		if f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, restored) != nil {
			return false
		}

		return restored.UID != deleted.UID
	}, managerTimeout, managerInterval, "deleting an owned ConfigMap did not trigger a reconcile that recreated it")

	assert.Equal(t, deleted.Data, f.getConfigMap(t, name).Data)
}

func testUpdatePropagation(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "update-propagation")
	f.assertReconcileSuccessfully(t)

	// Resource attributes are the DynaKube field to change here: they are mutable (the API URL is
	// not), only the gateway config renders them, and the watch predicate already treats a change
	// to them as relevant.
	t.Run("a gateway config change rolls the gateway pods and nothing else", func(t *testing.T) {
		before := f.resourceVersions(t)
		beforeHashes := f.configHashes(t)

		f.dk.Spec.ResourceAttributes = map[string]string{"deployment.environment": "integration"}
		require.NoError(t, clt.Update(t.Context(), f.dk))
		f.assertReconcileSuccessfully(t)

		after := f.resourceVersions(t)
		afterHashes := f.configHashes(t)

		gw := f.pm.Gateway().GetStatefulSetName()
		assert.NotEqual(t, before["ConfigMap/"+gw], after["ConfigMap/"+gw])
		assert.NotEqual(t, before["StatefulSet/"+gw], after["StatefulSet/"+gw])
		assert.NotEqual(t, beforeHashes["gateway"], afterHashes["gateway"], "the config hash must change so pods roll")

		assert.Equal(t, before["Service/"+gw], after["Service/"+gw], "the Service has no config dependency")
		assert.Equal(t, beforeHashes["scraper"], afterHashes["scraper"])
		assert.Equal(t, beforeHashes["targetallocator"], afterHashes["targetallocator"])
	})

	t.Run("a scraper poll interval change rolls only the scraper pods", func(t *testing.T) {
		beforeHashes := f.configHashes(t)

		f.pm.Spec.Scraper.TargetsPollInterval = new(metav1.Duration{Duration: 5 * time.Minute})
		f.updatePrometheusMonitoring(t)
		f.assertReconcileSuccessfully(t)

		afterHashes := f.configHashes(t)
		assert.NotEqual(t, beforeHashes["scraper"], afterHashes["scraper"])
		assert.Equal(t, beforeHashes["gateway"], afterHashes["gateway"])
		assert.Equal(t, beforeHashes["targetallocator"], afterHashes["targetallocator"])
	})

	t.Run("a target allocator scrape interval change rolls only the allocator pods", func(t *testing.T) {
		beforeHashes := f.configHashes(t)

		f.pm.Spec.TargetAllocator.ScrapeInterval = new(metav1.Duration{Duration: 90 * time.Second})
		f.updatePrometheusMonitoring(t)
		f.assertReconcileSuccessfully(t)

		afterHashes := f.configHashes(t)
		assert.NotEqual(t, beforeHashes["targetallocator"], afterHashes["targetallocator"])
		assert.Equal(t, beforeHashes["gateway"], afterHashes["gateway"])
		assert.Equal(t, beforeHashes["scraper"], afterHashes["scraper"])
	})

	// A replicas change is deliberately not covered here: each component's own lifecycle test
	// ("replicas change only touches the <workload>") already asserts that it leaves the ConfigMap
	// untouched, and the config hash is derived from that ConfigMap, so the pods cannot roll.
}

func testDriftCorrection(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "drift")
	f.assertReconcileSuccessfully(t)

	t.Run("a hand-edited image is reverted", func(t *testing.T) {
		for _, c := range components() {
			t.Run(c.name, func(t *testing.T) {
				w := c.getWorkload(t, f)
				setContainerImage(t, w.template, c.containerName, "evil.example.com/tampered:latest")
				require.NoError(t, clt.Update(t.Context(), w.object))

				f.assertReconcileSuccessfully(t)

				restored := c.getWorkload(t, f)
				assert.Equal(t, c.expectedImage, containerByName(t, restored.template, c.containerName).Image)
			})
		}
	})

	// A deleted ConfigMap is not covered here: testDriftCorrectionUnderManager deletes one and
	// proves both that it comes back and that the watch is what brought the reconcile, which is
	// strictly more than this test could show by calling Reconcile by hand.

	t.Run("a deleted Service is recreated", func(t *testing.T) {
		svc := f.getService(t, f.pm.TargetAllocator().GetDeploymentName())
		require.NoError(t, clt.Delete(t.Context(), svc))

		f.assertReconcileSuccessfully(t)

		restored := f.getService(t, f.pm.TargetAllocator().GetDeploymentName())
		assert.Equal(t, svc.Spec.Ports, restored.Spec.Ports)
		assert.Equal(t, svc.Spec.Selector, restored.Spec.Selector)
	})

	t.Run("a foreign label added by hand is preserved", func(t *testing.T) {
		// MergeInto deliberately keeps labels it does not own, so third-party tooling can label
		// the managed objects without the operator fighting it.
		const (
			foreignLabel = "foreign.example.com/owner"
			foreignOwner = "somebody-else"
		)

		for _, c := range components() {
			t.Run(c.name, func(t *testing.T) {
				name := c.objectName(f.pm)

				cm := f.getConfigMap(t, name)
				cm.Labels[foreignLabel] = foreignOwner
				require.NoError(t, clt.Update(t.Context(), cm))

				f.assertReconcileSuccessfully(t)

				assert.Equal(t, foreignOwner, f.getConfigMap(t, name).Labels[foreignLabel])
			})
		}
	})
}

// setContainerImage rewrites the image of one container in a pod template, so a drift test can put
// the workload back through the apiserver.
func setContainerImage(t *testing.T, tmpl *corev1.PodTemplateSpec, containerName, image string) {
	t.Helper()

	for i := range tmpl.Spec.Containers {
		if tmpl.Spec.Containers[i].Name == containerName {
			tmpl.Spec.Containers[i].Image = image

			return
		}
	}

	require.Failf(t, "container not found", "no container named %s in the pod template", containerName)
}

func testStatus(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "status")
	f.assertReconcileSuccessfully(t)

	stored := f.getStoredPrometheusMonitoring(t)

	t.Run("resolved images are recorded", func(t *testing.T) {
		assert.Equal(t, testCollectorImage, stored.Status.Gateway.ResolvedImage)
		assert.Equal(t, testCollectorImage, stored.Status.Scraper.ResolvedImage)
		assert.Equal(t, testTargetAllocatorImage, stored.Status.TargetAllocator.ResolvedImage)
	})

	t.Run("one availability condition per component", func(t *testing.T) {
		wantTypes := []string{
			prometheusmonitoring.GatewayAvailable,
			prometheusmonitoring.ScraperAvailable,
			prometheusmonitoring.TargetAllocatorAvailable,
		}

		gotTypes := make([]string, 0, len(stored.Status.Conditions))
		for _, condition := range stored.Status.Conditions {
			gotTypes = append(gotTypes, condition.Type)
		}

		assert.ElementsMatch(t, wantTypes, gotTypes)
	})

	// envtest has no kube-controller-manager, so the workloads never report ready replicas.
	// The reconcile itself succeeded, so every condition must be the "still rolling out" shape
	// rather than an error, and the derived phase must be Deploying. Reaching Running requires a
	// real cluster and is covered by the e2e follow-up.
	t.Run("components report as reconciling while no pods are ready", func(t *testing.T) {
		for _, condition := range stored.Status.Conditions {
			assert.Equal(t, metav1.ConditionFalse, condition.Status, condition.Type)
			assert.Equal(t, status.ReasonReconciling, condition.Reason, condition.Type)
			assert.NotEmpty(t, condition.Message, condition.Type)
		}

		assert.Equal(t, status.Deploying, stored.Status.Phase)
	})
}

// testErrorPaths covers the preconditions that make the reconcile bail out before it deploys
// anything. Which phase each one produces is already covered exhaustively and cheaply by
// TestReconcile and Test_setPhase against a fake client; what is only observable against a real
// apiserver is that the bail-out really happens before any object is written, so the phase is
// asserted here only to tie each precondition to the right outcome.
func testErrorPaths(t *testing.T, clt client.Client) {
	preconditions := []struct {
		name      string
		namespace string
		prepare   func(t *testing.T, f *fixture)
		phase     status.DeploymentPhase
	}{
		{
			// errDynaKubeNotFound is swallowed by setPhase: it is an expected transient state while
			// the DynaKube is still being created.
			name:      "missing DynaKube",
			namespace: "err-no-dynakube",
			prepare:   func(t *testing.T, f *fixture) { require.NoError(t, clt.Delete(t.Context(), f.dk)) },
			phase:     status.Deploying,
		},
		{
			name:      "DynaKube not running",
			namespace: "err-dynakube-pending",
			prepare: func(t *testing.T, f *fixture) {
				f.dk.Status.Phase = status.Deploying
				require.NoError(t, f.dk.UpdateStatus(t.Context(), clt))
			},
			phase: status.Deploying,
		},
		{
			name:      "token secret missing",
			namespace: "err-no-secret",
			prepare:   func(t *testing.T, f *fixture) { require.NoError(t, clt.Delete(t.Context(), f.getTokenSecret(t))) },
			phase:     status.Error,
		},
		{
			name:      "data-ingest key missing from the token secret",
			namespace: "err-no-data-ingest-key",
			prepare: func(t *testing.T, f *fixture) {
				secret := f.getTokenSecret(t)
				delete(secret.Data, token.DataIngestKey)
				require.NoError(t, clt.Update(t.Context(), secret))
			},
			phase: status.Error,
		},
	}

	for _, precondition := range preconditions {
		t.Run(precondition.name, func(t *testing.T) {
			f := newFixture(t, clt, precondition.namespace)
			precondition.prepare(t, f)

			require.NoError(t, f.reconcile(t))

			assert.Equal(t, precondition.phase, f.getStoredPrometheusMonitoring(t).Status.Phase)
			assert.Empty(t, f.listManagedObjects(t), "nothing may be deployed when the reconcile bails out")
		})
	}

	t.Run("objects created before a later failure are kept", func(t *testing.T) {
		// The reconcile loop breaks on the first error and nothing rolls back, so a component that
		// fails halfway leaves its earlier objects in place. Documenting the actual design: the
		// gateway is reconciled before the target allocator, so an allocator failure leaves a
		// complete gateway behind.
		f := newFixture(t, clt, "err-partial")
		f.assertReconcileSuccessfully(t)

		// A fleet management API that knows no target allocator image fails that component only.
		f.r = newReconcilerWithImages(t, clt, map[image.ComponentType]string{image.OTelCollector: testCollectorImage})

		require.ErrorContains(t, f.reconcile(t), "reconcile target allocator")

		gw := f.pm.Gateway().GetStatefulSetName()
		assert.Contains(t, f.listManagedObjects(t), managedObject{"StatefulSet", gw})
		assert.Equal(t, status.Error, f.getStoredPrometheusMonitoring(t).Status.Phase)
	})
}

// Deletion has no test of its own. The two things it could assert are covered elsewhere and
// cheaper: that reconciling a deleted PrometheusMonitoring is a no-op is TestReconcile's
// "get prometheusmonitoring deleted" case against a fake client, and that nothing holds the object
// back is the finalizer check in testCreation. The cascade itself needs a real garbage collector,
// so it belongs to the e2e suite, which already asserts the workloads disappear after a delete.

// callCounter wraps a client.Client to count the writes issued through it. Status subresource
// writes go through SubResource(), which the embedded client serves directly, so they are
// deliberately not counted here.
type callCounter struct {
	client.Client

	creates int
	updates int
	patches int
	deletes int
}

func (c *callCounter) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.creates++

	return c.Client.Create(ctx, obj, opts...)
}

func (c *callCounter) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	c.updates++

	return c.Client.Update(ctx, obj, opts...)
}

func (c *callCounter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	c.patches++

	return c.Client.Patch(ctx, obj, patch, opts...)
}

func (c *callCounter) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	c.deletes++

	return c.Client.Delete(ctx, obj, opts...)
}

// fixture is one fully provisioned PrometheusMonitoring scenario in its own namespace.
type fixture struct {
	clt client.Client
	r   *Reconciler
	pm  *prometheusmonitoring.PrometheusMonitoring
	dk  *dynakube.DynaKube
	ns  string
}

// newFixture creates a namespace, a token Secret, a Running DynaKube and a default
// PrometheusMonitoring, plus a Reconciler whose Dynatrace client factory is stubbed out. No token
// beyond the two the reconcile reads, and no image in the spec: the images are resolved from the
// stubbed fleet management API, the way they are for a user who sets none.
func newFixture(t *testing.T, clt client.Client, name string) *fixture {
	t.Helper()

	ns := "pm-" + name
	integrationtests.CreateNamespace(t, clt, ns)

	integrationtests.CreateKubernetesObject(t, clt, &corev1.Secret{
		Name:      testDynaKubeName,
		Namespace: ns,
		Data: map[string][]byte{
			token.APIKey:        []byte(testAPIToken),
			token.DataIngestKey: []byte(testDataIngestToken),
		},
	})

	dk := &dynakube.DynaKube{
		Name:      testDynaKubeName,
		Namespace: ns,
		Spec:      dynakube.DynaKubeSpec{APIURL: testAPIURL},
		Status:    dynakube.DynaKubeStatus{Phase: status.Running, KubernetesClusterName: testClusterName},
	}
	integrationtests.CreateDynakube(t, clt, dk)

	pm := createDefaultedPrometheusMonitoring(t, clt, ns)

	return &fixture{clt: clt, r: newStubbedReconciler(t, clt), pm: pm, dk: dk, ns: ns}
}

// createDefaultedPrometheusMonitoring creates the PrometheusMonitoring as an unstructured object carrying only the
// fields a user would actually write, so the API server applies every CRD default before the
// reconciler ever sees it. Creating it from the typed struct would send explicit zero values for
// metav1.Duration fields (which marshal to "0s" rather than being omitted) and the defaults would
// silently not apply.
func createDefaultedPrometheusMonitoring(t *testing.T, clt client.Client, ns string) *prometheusmonitoring.PrometheusMonitoring {
	t.Helper()

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(v1alpha1.GroupVersion.WithKind("PrometheusMonitoring"))
	obj.SetName("pm")
	obj.SetNamespace(ns)

	require.NoError(t, unstructured.SetNestedMap(obj.Object, map[string]any{
		"dynaKubeName": testDynaKubeName,
	}, "spec"))

	integrationtests.CreateKubernetesObject(t, clt, obj)

	pm := &prometheusmonitoring.PrometheusMonitoring{}
	require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: obj.GetName(), Namespace: ns}, pm))

	return pm
}

// newStubbedReconciler is the production reconciler with only the Dynatrace API client factory
// replaced, so no test reaches out to a tenant. Its image client answers the way fleet management
// does, which is where the components' images come from when the spec names none. Gateway and
// scraper are both plain OTel Collectors, so they share one component type and therefore one image.
func newStubbedReconciler(t *testing.T, clt client.Client) *Reconciler {
	t.Helper()

	return newReconcilerWithImages(t, clt, map[image.ComponentType]string{
		image.OTelCollector:   testCollectorImage,
		image.TargetAllocator: testTargetAllocatorImage,
	})
}

// newReconcilerWithImages is newStubbedReconciler with the fleet management answers spelled out, so
// a test can leave a component's image out and make that component's reconcile fail.
func newReconcilerWithImages(t *testing.T, clt client.Client, images map[image.ComponentType]string) *Reconciler {
	t.Helper()

	imageClient := imagemock.NewClient(t)
	imageClient.EXPECT().GetComponentLatestInfo(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, component image.ComponentType, _ string) (*image.Info, error) {
			uri, known := images[component]
			if !known {
				return nil, fmt.Errorf("no image for component %s", component)
			}

			return &image.Info{URI: uri}, nil
		}).Maybe()

	r := NewReconciler(clt)
	r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, string, time.Duration) (*dynatrace.Client, error) {
		return &dynatrace.Client{Images: imageClient}, nil
	}

	return r
}

func (f *fixture) request() ctrl.Request {
	return ctrl.Request{Name: f.pm.Name, Namespace: f.ns}
}

func (f *fixture) reconcile(t *testing.T) error {
	t.Helper()

	_, err := f.r.Reconcile(t.Context(), f.request())

	return err
}

func (f *fixture) assertReconcileSuccessfully(t *testing.T) {
	t.Helper()

	require.NoError(t, f.reconcile(t))
}

// updatePrometheusMonitoring persists spec changes made on the in-memory object. The reconciler reads the
// PrometheusMonitoring from the API server, so a spec change that is not written has no effect.
func (f *fixture) updatePrometheusMonitoring(t *testing.T) {
	t.Helper()

	stored := f.getStoredPrometheusMonitoring(t)
	stored.Spec = f.pm.Spec
	require.NoError(t, f.clt.Update(t.Context(), stored))

	f.pm.ResourceVersion = stored.ResourceVersion
}

func (f *fixture) getStoredPrometheusMonitoring(t *testing.T) *prometheusmonitoring.PrometheusMonitoring {
	t.Helper()

	stored := &prometheusmonitoring.PrometheusMonitoring{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKeyFromObject(f.pm), stored))

	return stored
}

func (f *fixture) getTokenSecret(t *testing.T) *corev1.Secret {
	t.Helper()

	secret := &corev1.Secret{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKey{Name: f.dk.Tokens(), Namespace: f.ns}, secret))

	return secret
}

func (f *fixture) getConfigMap(t *testing.T, name string) *corev1.ConfigMap {
	t.Helper()

	cm := &corev1.ConfigMap{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, cm))

	return cm
}

func (f *fixture) getService(t *testing.T, name string) *corev1.Service {
	t.Helper()

	svc := &corev1.Service{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, svc))

	return svc
}

func (f *fixture) getDeployment(t *testing.T, name string) *appsv1.Deployment {
	t.Helper()

	deploy := &appsv1.Deployment{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, deploy))

	return deploy
}

func (f *fixture) getGatewayStatefulSet(t *testing.T) *appsv1.StatefulSet {
	t.Helper()

	sts := &appsv1.StatefulSet{}
	require.NoError(t, f.clt.Get(t.Context(),
		client.ObjectKey{Name: f.pm.Gateway().GetStatefulSetName(), Namespace: f.ns}, sts))

	return sts
}

func (f *fixture) getManagedObject(t *testing.T, want managedObject) client.Object {
	t.Helper()

	key := client.ObjectKey{Name: want.name, Namespace: f.ns}

	var obj client.Object

	switch want.kind {
	case "ConfigMap":
		obj = &corev1.ConfigMap{}
	case "Service":
		obj = &corev1.Service{}
	case "Deployment":
		obj = &appsv1.Deployment{}
	case "StatefulSet":
		obj = &appsv1.StatefulSet{}
	default:
		require.Failf(t, "unhandled kind", "no getter for %s", want.kind)
	}

	require.NoError(t, f.clt.Get(t.Context(), key, obj))

	return obj
}

// observedObjectKinds maps every kind the inventory checks look at to an empty list to read it
// with. Looking at more kinds than the controller actually creates is deliberate: a component that
// starts creating a PodDisruptionBudget, a NetworkPolicy or its own RBAC shows up as a diff instead
// of going unnoticed.
func observedObjectKinds() map[string]client.ObjectList {
	return map[string]client.ObjectList{
		"ConfigMap":           &corev1.ConfigMapList{},
		"Service":             &corev1.ServiceList{},
		"Secret":              &corev1.SecretList{},
		"ServiceAccount":      &corev1.ServiceAccountList{},
		"Deployment":          &appsv1.DeploymentList{},
		"StatefulSet":         &appsv1.StatefulSetList{},
		"DaemonSet":           &appsv1.DaemonSetList{},
		"PodDisruptionBudget": &policyv1.PodDisruptionBudgetList{},
		"NetworkPolicy":       &networkingv1.NetworkPolicyList{},
		"Role":                &rbacv1.RoleList{},
		"RoleBinding":         &rbacv1.RoleBindingList{},
	}
}

// fixtureObjects is what newFixture itself puts into the namespace, in the kinds
// observedObjectKinds covers. The DynaKube and the PrometheusMonitoring are not among them.
func fixtureObjects() []managedObject {
	return []managedObject{{"Secret", testDynaKubeName}}
}

// listManagedObjects returns every object in the fixture namespace that carries the operator's
// managed-by label.
func (f *fixture) listManagedObjects(t *testing.T) []managedObject {
	t.Helper()

	return f.listObjects(t,
		client.InNamespace(f.ns),
		client.MatchingLabels{k8slabel.AppManagedByLabel: version.AppName})
}

// listAllObjects returns every object in the fixture namespace regardless of labels, which is what
// makes the inventory check independent of the operator labeling its objects correctly.
func (f *fixture) listAllObjects(t *testing.T) []managedObject {
	t.Helper()

	return f.listObjects(t, client.InNamespace(f.ns))
}

func (f *fixture) listObjects(t *testing.T, opts ...client.ListOption) []managedObject {
	t.Helper()

	found := make([]managedObject, 0)

	for kind, list := range observedObjectKinds() {
		require.NoErrorf(t, f.clt.List(t.Context(), list, opts...), "cannot list %ss", kind)

		require.NoError(t, apimeta.EachListItem(list, func(obj runtime.Object) error {
			accessor, err := apimeta.Accessor(obj)
			if err != nil {
				return err
			}

			found = append(found, managedObject{kind, accessor.GetName()})

			return nil
		}))
	}

	return found
}

// resourceVersions maps "Kind/name" to the stored resourceVersion of every managed object.
func (f *fixture) resourceVersions(t *testing.T) map[string]string {
	t.Helper()

	versions := make(map[string]string)
	for _, want := range expectedManagedObjects(f.pm) {
		versions[want.String()] = f.getManagedObject(t, want).GetResourceVersion()
	}

	return versions
}

// configHashes maps a component name to the config-hash annotation on its pod template. A change
// there is what actually rolls the pods when a ConfigMap's content changes.
//
// Each component names its annotation after itself (gateway-config-hash, scraper-config-hash,
// allocator-config-hash) and the constants are package private, so the annotation is found by
// suffix. Exactly one match is required: picking the last match out of a map range would depend on
// the iteration order the moment a component grew a second hash annotation.
func (f *fixture) configHashes(t *testing.T) map[string]string {
	t.Helper()

	hashes := make(map[string]string, len(components()))

	for _, c := range components() {
		tmpl := c.getWorkload(t, f).template

		matches := make([]string, 0, 1)

		for key, value := range tmpl.Annotations {
			if strings.HasSuffix(key, "-config-hash") {
				matches = append(matches, value)
			}
		}

		require.Lenf(t, matches, 1, "%s pod template must carry exactly one config hash annotation, has %v",
			c.name, tmpl.Annotations)
		require.NotEmptyf(t, matches[0], "%s config hash annotation is empty", c.name)

		hashes[c.name] = matches[0]
	}

	return hashes
}

func (f *fixture) assertVolumeSourceResolves(t *testing.T, volume corev1.Volume) {
	t.Helper()

	if cm := volume.ConfigMap; cm != nil {
		f.assertConfigMapKeys(t, cm.Name, keyPaths(cm.Items))
	}

	if projected := volume.Projected; projected != nil {
		for _, source := range projected.Sources {
			if source.ConfigMap != nil {
				f.assertConfigMapKeys(t, source.ConfigMap.Name, keyPaths(source.ConfigMap.Items))
			}

			if source.Secret != nil {
				f.assertSecretKeys(t, source.Secret.Name, keyPaths(source.Secret.Items))
			}
		}
	}

	if secret := volume.Secret; secret != nil {
		f.assertSecretKeys(t, secret.SecretName, keyPaths(secret.Items))
	}
}

func keyPaths(items []corev1.KeyToPath) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}

	return keys
}

func (f *fixture) assertEnvSourceResolves(t *testing.T, env corev1.EnvVar) {
	t.Helper()

	if env.ValueFrom == nil {
		return
	}

	if ref := env.ValueFrom.SecretKeyRef; ref != nil {
		f.assertSecretKeys(t, ref.Name, []string{ref.Key})
	}

	if ref := env.ValueFrom.ConfigMapKeyRef; ref != nil {
		f.assertConfigMapKeys(t, ref.Name, []string{ref.Key})
	}
}

func (f *fixture) assertConfigMapKeys(t *testing.T, name string, keys []string) {
	t.Helper()

	cm := &corev1.ConfigMap{}
	require.NoErrorf(t, f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, cm),
		"referenced ConfigMap %s does not exist", name)

	for _, key := range keys {
		assert.Containsf(t, cm.Data, key, "ConfigMap %s has no key %s", name, key)
	}
}

func (f *fixture) assertSecretKeys(t *testing.T, name string, keys []string) {
	t.Helper()

	f.assertSecretExists(t, name)

	secret := &corev1.Secret{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, secret))

	for _, key := range keys {
		assert.Containsf(t, secret.Data, key, "Secret %s has no key %s", name, key)
	}
}

// assertSecretExists is the weaker check for Secrets that are referenced by name only, where no
// particular key is expected (an image pull secret, for example).
func (f *fixture) assertSecretExists(t *testing.T, name string) {
	t.Helper()

	require.NoErrorf(t, f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, &corev1.Secret{}),
		"referenced Secret %s does not exist", name)
}
