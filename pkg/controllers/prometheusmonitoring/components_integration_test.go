// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
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
	"k8s.io/apimachinery/pkg/types"
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
	testPaaSToken       = "dt0c01.INTEGRATIONPAASTOKENVALUE"

	testGatewayImage         = "registry.example.com/dynatrace/gateway:1.2.3"
	testScraperImage         = "registry.example.com/dynatrace/scraper:1.2.3"
	testTargetAllocatorImage = "registry.example.com/dynatrace/target-allocator:1.2.3"

	testDynaKubeName = "dk"
	testAPIURL       = "https://tenant.dev.dynatracelabs.com/api"

	// How long to wait on a live Manager to notice something and act on it.
	managerTimeout  = 30 * time.Second
	managerInterval = 100 * time.Millisecond
)

// component describes one deployed PrometheusMonitoring component so that the aspect tests below can be
// table driven. Adding a component later (for example Self Monitoring, which is intentionally not
// covered yet) means adding one entry here plus its expected object set.
type component struct {
	name string
	// workload is the Deployment or StatefulSet, and also the ConfigMap and Service name:
	// all objects of a component share one name.
	objectName func(pm *prometheusmonitoring.PrometheusMonitoring) string
	labels     func() *k8slabel.Labels
	// podSpec returns the pod template of the component's workload.
	podSpec       func(t *testing.T, f *fixture) (*corev1.PodTemplateSpec, *metav1.LabelSelector)
	containerName string
	// hasService is false for the scraper: nothing connects to it inbound.
	hasService bool
	// serviceAccount is created by the helm chart, not by the operator.
	serviceAccount string
	// clusterRoleTemplate and clusterRoleBindingTemplate are the component's own RBAC in the helm
	// chart, relative to config/helm/chart/default/templates/Common.
	clusterRoleTemplate        string
	clusterRoleBindingTemplate string
}

func components() []component {
	return []component{
		{
			name:                       "gateway",
			objectName:                 func(pm *prometheusmonitoring.PrometheusMonitoring) string { return pm.Gateway().GetStatefulSetName() },
			labels:                     func() *k8slabel.Labels { return k8slabel.New("opentelemetry-gateway", "otel-gateway", "") },
			podSpec:                    gatewayPodSpec,
			containerName:              "gateway",
			hasService:                 true,
			serviceAccount:             "dynatrace-prometheus-gateway",
			clusterRoleTemplate:        "prometheus/clusterrole-prometheus-gateway.yaml",
			clusterRoleBindingTemplate: "prometheus/clusterrolebinding-prometheus-gateway.yaml",
		},
		{
			name:                       "scraper",
			objectName:                 func(pm *prometheusmonitoring.PrometheusMonitoring) string { return pm.Scraper().GetDeploymentName() },
			labels:                     k8slabel.OTelScraper,
			podSpec:                    scraperPodSpec,
			containerName:              "scraper",
			hasService:                 false,
			serviceAccount:             "dynatrace-prometheus-scraper",
			clusterRoleTemplate:        "prometheus/clusterrole-prometheus-scraper.yaml",
			clusterRoleBindingTemplate: "prometheus/clusterrolebinding-prometheus-scraper.yaml",
		},
		{
			name: "targetallocator",
			objectName: func(pm *prometheusmonitoring.PrometheusMonitoring) string {
				return pm.TargetAllocator().GetDeploymentName()
			},
			labels:                     k8slabel.OTelTargetAllocator,
			podSpec:                    targetAllocatorPodSpec,
			containerName:              "targetallocator",
			hasService:                 true,
			serviceAccount:             "dynatrace-target-allocator",
			clusterRoleTemplate:        "prometheus/clusterrole-target-allocator.yaml",
			clusterRoleBindingTemplate: "prometheus/clusterrolebinding-target-allocator.yaml",
		},
	}
}

func gatewayPodSpec(t *testing.T, f *fixture) (*corev1.PodTemplateSpec, *metav1.LabelSelector) {
	t.Helper()

	sts := f.gatewayStatefulSet(t)

	return &sts.Spec.Template, sts.Spec.Selector
}

func scraperPodSpec(t *testing.T, f *fixture) (*corev1.PodTemplateSpec, *metav1.LabelSelector) {
	t.Helper()

	deploy := f.deployment(t, f.pm.Scraper().GetDeploymentName())

	return &deploy.Spec.Template, deploy.Spec.Selector
}

func targetAllocatorPodSpec(t *testing.T, f *fixture) (*corev1.PodTemplateSpec, *metav1.LabelSelector) {
	t.Helper()

	deploy := f.deployment(t, f.pm.TargetAllocator().GetDeploymentName())

	return &deploy.Spec.Template, deploy.Spec.Selector
}

// TestReconcileComponents drives the full top-level Reconcile against a real API server.
// One envtest control plane is shared by all subtests; each subtest gets its own namespace so the
// object sets never overlap.
func TestReconcileComponents(t *testing.T) {
	clt, cfg := integrationtests.SetupTestEnvironmentWithConfig(t)

	t.Run("creation", func(t *testing.T) { testCreation(t, clt) })
	t.Run("ownership", func(t *testing.T) { testOwnership(t, clt) })
	t.Run("workload-spec", func(t *testing.T) { testWorkloadSpec(t, clt) })
	t.Run("references-resolve", func(t *testing.T) { testReferencesResolve(t, clt) })
	t.Run("wiring", func(t *testing.T) { testWiring(t, clt) })
	t.Run("configmaps", func(t *testing.T) { testConfigMaps(t, clt) })
	t.Run("secrets", func(t *testing.T) { testSecrets(t, clt) })
	t.Run("customization", func(t *testing.T) { testCustomization(t, clt) })
	t.Run("idempotency", func(t *testing.T) { testIdempotency(t, clt) })
	t.Run("update-propagation", func(t *testing.T) { testUpdatePropagation(t, clt) })
	t.Run("drift-correction", func(t *testing.T) { testDriftCorrection(t, clt) })
	t.Run("status", func(t *testing.T) { testStatus(t, clt) })
	t.Run("error-paths", func(t *testing.T) { testErrorPaths(t, clt) })
	t.Run("deletion", func(t *testing.T) { testDeletion(t, clt) })
	t.Run("operator-rbac", func(t *testing.T) { testOperatorRBAC(t, clt, cfg) })
	t.Run("component-rbac", func(t *testing.T) { testComponentRBAC(t, clt, cfg) })
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
	f.reconcileSuccessfully(t)

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
}

func testOwnership(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "ownership")
	f.reconcileSuccessfully(t)

	for _, want := range expectedManagedObjects(f.pm) {
		t.Run(want.String(), func(t *testing.T) {
			obj := f.getManagedObject(t, want)

			refs := obj.GetOwnerReferences()
			require.Len(t, refs, 1)

			ref := refs[0]
			assert.Equal(t, "PrometheusMonitoring", ref.Kind)
			assert.Equal(t, v1alpha1.GroupVersion.String(), ref.APIVersion)
			assert.Equal(t, f.pm.Name, ref.Name)
			assert.Equal(t, f.pm.UID, ref.UID)
			assert.Equal(t, new(true), ref.Controller)
			assert.Equal(t, new(true), ref.BlockOwnerDeletion)
		})
	}
}

func testWorkloadSpec(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "workload-spec")
	f.reconcileSuccessfully(t)

	for _, c := range components() {
		t.Run(c.name, func(t *testing.T) { assertWorkloadSpec(t, f, c) })
	}

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

	tmpl, selector := c.podSpec(t, f)

	t.Run("selector matches the pod template labels", func(t *testing.T) {
		require.NotNil(t, selector)
		assert.NotEmpty(t, selector.MatchLabels)

		for key, value := range selector.MatchLabels {
			assert.Equal(t, value, tmpl.Labels[key], "pod template is missing selector label %s", key)
		}

		assert.Equal(t, c.labels().AsSelector(), selector.MatchLabels)
	})

	t.Run("service account and token automounting", func(t *testing.T) {
		assert.Equal(t, c.serviceAccount, tmpl.Spec.ServiceAccountName)
		assert.Equal(t, new(true), tmpl.Spec.AutomountServiceAccountToken)
	})

	t.Run("container hardening", func(t *testing.T) {
		assertContainerHardened(t, containerByName(t, tmpl, c.containerName))
	})

	t.Run("probes", func(t *testing.T) {
		container := containerByName(t, tmpl, c.containerName)
		require.NotNil(t, container.LivenessProbe)
		require.NotNil(t, container.ReadinessProbe)
		assert.NotNil(t, container.LivenessProbe.HTTPGet)
		assert.NotNil(t, container.ReadinessProbe.HTTPGet)
	})

	t.Run("image", func(t *testing.T) {
		assert.Equal(t, f.imageFor(c.name), containerByName(t, tmpl, c.containerName).Image)
	})
}

func assertContainerHardened(t *testing.T, container corev1.Container) {
	t.Helper()

	sec := container.SecurityContext
	require.NotNil(t, sec)

	assert.Equal(t, new(false), sec.Privileged)
	assert.Equal(t, new(false), sec.AllowPrivilegeEscalation)
	assert.Equal(t, new(true), sec.RunAsNonRoot)
	assert.Equal(t, new(true), sec.ReadOnlyRootFilesystem)
	require.NotNil(t, sec.RunAsUser)
	assert.Positive(t, *sec.RunAsUser)
	require.NotNil(t, sec.SeccompProfile)
	assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, sec.SeccompProfile.Type)
	require.NotNil(t, sec.Capabilities)
	assert.Equal(t, []corev1.Capability{"ALL"}, sec.Capabilities.Drop)
}

func assertServiceRoutesToWorkload(t *testing.T, f *fixture, c component) {
	t.Helper()

	tmpl, _ := c.podSpec(t, f)
	container := containerByName(t, tmpl, c.containerName)
	svc := f.service(t, c.objectName(f.pm))

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
	f.reconcileSuccessfully(t)

	for _, c := range components() {
		t.Run(c.name, func(t *testing.T) {
			tmpl, _ := c.podSpec(t, f)

			for _, volume := range tmpl.Spec.Volumes {
				f.assertVolumeSourceResolves(t, volume)
			}

			container := containerByName(t, tmpl, c.containerName)
			for _, env := range container.Env {
				f.assertEnvSourceResolves(t, env)
			}

			// A pull secret is referenced by name only, so a missing one is not a config error the
			// operator can see: the pods just sit in ImagePullBackOff.
			for _, pullSecret := range tmpl.Spec.ImagePullSecrets {
				f.assertSecretExists(t, pullSecret.Name)
			}

			t.Run("every volume mount has a volume", func(t *testing.T) {
				for _, mount := range container.VolumeMounts {
					assert.True(t, hasVolume(tmpl.Spec.Volumes, mount.Name),
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

func testCustomization(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "customization")

	f.pm.Spec.Gateway.Replicas = new(int32(3))
	f.pm.Spec.Gateway.ImagePullPolicy = corev1.PullAlways
	f.pm.Spec.Gateway.Labels = map[string]string{"custom-label": "gateway"}
	f.pm.Spec.Gateway.Annotations = map[string]string{"custom-annotation": "gateway"}
	f.pm.Spec.Gateway.PriorityClassName = "system-cluster-critical"
	f.pm.Spec.Gateway.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}
	f.pm.Spec.Gateway.Tolerations = []corev1.Toleration{{Key: "dedicated", Operator: corev1.TolerationOpExists}}
	f.pm.Spec.Scraper.Args = []string{"--feature-gates=+foo"}
	f.pm.Spec.Scraper.Replicas = new(int32(4))
	f.pm.Spec.TargetAllocator.Replicas = new(int32(2))
	f.updatePrometheusMonitoring(t)
	f.reconcileSuccessfully(t)

	t.Run("gateway", func(t *testing.T) {
		sts := f.gatewayStatefulSet(t)
		assert.Equal(t, new(int32(3)), sts.Spec.Replicas)
		assert.Equal(t, "gateway", sts.Spec.Template.Labels["custom-label"])
		assert.Equal(t, "gateway", sts.Spec.Template.Annotations["custom-annotation"])
		assert.Equal(t, "system-cluster-critical", sts.Spec.Template.Spec.PriorityClassName)
		assert.Equal(t, map[string]string{"kubernetes.io/os": "linux"}, sts.Spec.Template.Spec.NodeSelector)
		assert.Equal(t, []corev1.Toleration{{Key: "dedicated", Operator: corev1.TolerationOpExists}},
			sts.Spec.Template.Spec.Tolerations)

		container := containerByName(t, &sts.Spec.Template, "gateway")
		assert.Equal(t, corev1.PullAlways, container.ImagePullPolicy)

		// Operator-owned labels must survive user labels, otherwise the selector stops matching.
		for key, value := range k8slabel.New("opentelemetry-gateway", "otel-gateway", "").AsSelector() {
			assert.Equal(t, value, sts.Spec.Template.Labels[key])
		}
	})

	t.Run("scraper", func(t *testing.T) {
		deploy := f.deployment(t, f.pm.Scraper().GetDeploymentName())
		assert.Equal(t, new(int32(4)), deploy.Spec.Replicas)

		container := containerByName(t, &deploy.Spec.Template, "scraper")
		require.NotEmpty(t, container.Args)
		assert.Equal(t, "--config=/conf/scraper.yaml", container.Args[0], "the operator-managed config flag must come first")
		assert.Contains(t, container.Args, "--feature-gates=+foo")
	})

	t.Run("target allocator", func(t *testing.T) {
		deploy := f.deployment(t, f.pm.TargetAllocator().GetDeploymentName())
		assert.Equal(t, new(int32(2)), deploy.Spec.Replicas)
	})

	// Each component's own lifecycle test covers the partial merge, where only a rollingUpdate
	// block is set and the apiserver's defaulted type survives. Two things are only observable from
	// here: that one reconcile routes each of the three strategies to the right workload (they are
	// three different fields on two different kinds), and the other branch of the merge helpers,
	// where an explicit non-rolling type has to clear the rollingUpdate block the apiserver
	// defaulted in, because the apiserver rejects the two together.
	t.Run("all three update strategies are honored", func(t *testing.T) {
		f.pm.Spec.Gateway.UpdateStrategy = appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType}
		f.pm.Spec.TargetAllocator.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		f.pm.Spec.Scraper.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		f.updatePrometheusMonitoring(t)
		f.reconcileSuccessfully(t)

		gatewayStrategy := f.gatewayStatefulSet(t).Spec.UpdateStrategy
		assert.Equal(t, appsv1.OnDeleteStatefulSetStrategyType, gatewayStrategy.Type)
		assert.Nil(t, gatewayStrategy.RollingUpdate, "OnDelete must not keep a rollingUpdate block")

		for name, workload := range map[string]string{
			"target allocator": f.pm.TargetAllocator().GetDeploymentName(),
			"scraper":          f.pm.Scraper().GetDeploymentName(),
		} {
			strategy := f.deployment(t, workload).Spec.Strategy
			assert.Equal(t, appsv1.RecreateDeploymentStrategyType, strategy.Type, name)
			assert.Nil(t, strategy.RollingUpdate, "%s: Recreate must not keep a rollingUpdate block", name)
		}
	})
}

// testIdempotency asserts that repeating a reconcile changes nothing. Each component's own
// lifecycle test already covers that for its own objects; what is only visible from here is the
// top-level reconciler, which writes the PrometheusMonitoring status on every single pass.
func testIdempotency(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "idempotency")
	f.reconcileSuccessfully(t)

	before := f.resourceVersions(t)
	beforeOwner := f.storedPrometheusMonitoring(t).ResourceVersion

	// resourceVersion alone is not proof: the API server re-defaults an incoming object before
	// comparing it to storage, so a reconcile that sends a stale object still issues a real write
	// that is silently no-op'ed server-side. Counting the calls the reconcile makes catches that,
	// for every kind of write rather than only for Update: a component switching from Update to
	// Patch, or deleting and recreating an object it could have updated, would otherwise slip
	// through while still restarting pods on a real cluster.
	counting := &callCounter{Client: clt}
	f.r = newStubbedReconciler(t, counting)

	for range 3 {
		f.reconcileSuccessfully(t)
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
	assert.Equal(t, beforeOwner, f.storedPrometheusMonitoring(t).ResourceVersion,
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

	deleted := f.configMap(t, name)
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

	assert.Equal(t, deleted.Data, f.configMap(t, name).Data)
}

func testUpdatePropagation(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "update-propagation")
	f.reconcileSuccessfully(t)

	t.Run("a gateway config change rolls the gateway pods and nothing else", func(t *testing.T) {
		before := f.resourceVersions(t)
		beforeHashes := f.configHashes(t)

		f.dk.Spec.APIURL = "https://other.dev.dynatracelabs.com/api"
		require.NoError(t, clt.Update(t.Context(), f.dk))
		f.reconcileSuccessfully(t)

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
		f.reconcileSuccessfully(t)

		afterHashes := f.configHashes(t)
		assert.NotEqual(t, beforeHashes["scraper"], afterHashes["scraper"])
		assert.Equal(t, beforeHashes["gateway"], afterHashes["gateway"])
		assert.Equal(t, beforeHashes["targetallocator"], afterHashes["targetallocator"])
	})

	t.Run("a target allocator scrape interval change rolls only the allocator pods", func(t *testing.T) {
		beforeHashes := f.configHashes(t)

		f.pm.Spec.TargetAllocator.ScrapeInterval = new(metav1.Duration{Duration: 90 * time.Second})
		f.updatePrometheusMonitoring(t)
		f.reconcileSuccessfully(t)

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
	f.reconcileSuccessfully(t)

	t.Run("a hand-edited image is reverted", func(t *testing.T) {
		sts := f.gatewayStatefulSet(t)
		sts.Spec.Template.Spec.Containers[0].Image = "evil.example.com/tampered:latest"
		require.NoError(t, clt.Update(t.Context(), sts))

		f.reconcileSuccessfully(t)

		assert.Equal(t, testGatewayImage, f.gatewayStatefulSet(t).Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("a deleted ConfigMap is recreated", func(t *testing.T) {
		cm := f.configMap(t, f.pm.Scraper().GetDeploymentName())
		require.NoError(t, clt.Delete(t.Context(), cm))

		f.reconcileSuccessfully(t)

		restored := f.configMap(t, f.pm.Scraper().GetDeploymentName())
		assert.Equal(t, cm.Data, restored.Data)
		assert.True(t, metav1.IsControlledBy(restored, f.pm))
	})

	t.Run("a deleted Service is recreated", func(t *testing.T) {
		svc := f.service(t, f.pm.TargetAllocator().GetDeploymentName())
		require.NoError(t, clt.Delete(t.Context(), svc))

		f.reconcileSuccessfully(t)

		restored := f.service(t, f.pm.TargetAllocator().GetDeploymentName())
		assert.Equal(t, svc.Spec.Ports, restored.Spec.Ports)
		assert.Equal(t, svc.Spec.Selector, restored.Spec.Selector)
	})

	t.Run("a foreign label added by hand is preserved", func(t *testing.T) {
		// MergeInto deliberately keeps labels it does not own, so third-party tooling can label
		// the managed objects without the operator fighting it.
		cm := f.configMap(t, f.pm.Gateway().GetStatefulSetName())
		cm.Labels["foreign.example.com/owner"] = "somebody-else"
		require.NoError(t, clt.Update(t.Context(), cm))

		f.reconcileSuccessfully(t)

		assert.Equal(t, "somebody-else",
			f.configMap(t, f.pm.Gateway().GetStatefulSetName()).Labels["foreign.example.com/owner"])
	})
}

func testStatus(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "status")
	f.reconcileSuccessfully(t)

	stored := f.storedPrometheusMonitoring(t)

	t.Run("resolved images are recorded", func(t *testing.T) {
		assert.Equal(t, testGatewayImage, stored.Status.Gateway.ResolvedImage)
		assert.Equal(t, testScraperImage, stored.Status.Scraper.ResolvedImage)
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

func testErrorPaths(t *testing.T, clt client.Client) {
	t.Run("missing DynaKube", func(t *testing.T) {
		f := newFixture(t, clt, "err-no-dynakube")
		require.NoError(t, clt.Delete(t.Context(), f.dk))

		// errDynaKubeNotFound is swallowed by setPhase: it is an expected transient state while the
		// DynaKube is still being created.
		require.NoError(t, f.reconcile(t))

		assert.Equal(t, status.Deploying, f.storedPrometheusMonitoring(t).Status.Phase)
		assert.Empty(t, f.listManagedObjects(t), "nothing may be deployed without a DynaKube")
	})

	t.Run("DynaKube not running", func(t *testing.T) {
		f := newFixture(t, clt, "err-dynakube-pending")
		f.dk.Status.Phase = status.Deploying
		require.NoError(t, f.dk.UpdateStatus(t.Context(), clt))

		require.NoError(t, f.reconcile(t))

		assert.Equal(t, status.Deploying, f.storedPrometheusMonitoring(t).Status.Phase)
		assert.Empty(t, f.listManagedObjects(t))
	})

	t.Run("token secret missing", func(t *testing.T) {
		f := newFixture(t, clt, "err-no-secret")
		require.NoError(t, clt.Delete(t.Context(), f.tokenSecret(t)))

		require.NoError(t, f.reconcile(t))

		assert.Equal(t, status.Error, f.storedPrometheusMonitoring(t).Status.Phase)
		assert.Empty(t, f.listManagedObjects(t))
	})

	t.Run("data-ingest key missing from the token secret", func(t *testing.T) {
		f := newFixture(t, clt, "err-no-data-ingest-key")
		secret := f.tokenSecret(t)
		delete(secret.Data, token.DataIngestKey)
		require.NoError(t, clt.Update(t.Context(), secret))

		require.NoError(t, f.reconcile(t))

		assert.Equal(t, status.Error, f.storedPrometheusMonitoring(t).Status.Phase)
		assert.Empty(t, f.listManagedObjects(t), "the gateway must not be deployed without a data-ingest token")
	})

	t.Run("objects created before a later failure are kept", func(t *testing.T) {
		// The reconcile loop breaks on the first error and nothing rolls back, so a component that
		// fails halfway leaves its earlier objects in place. Documenting the actual design: the
		// gateway is reconciled first, so a scraper failure leaves a complete gateway behind.
		f := newFixture(t, clt, "err-partial")
		f.reconcileSuccessfully(t)

		f.pm.Spec.Scraper.Image = ""
		f.updatePrometheusMonitoring(t)

		// Without an image and without a fleet image client, the scraper reconcile fails.
		require.Error(t, f.reconcile(t))

		gw := f.pm.Gateway().GetStatefulSetName()
		assert.Contains(t, f.listManagedObjects(t), managedObject{"StatefulSet", gw})
		assert.Equal(t, status.Error, f.storedPrometheusMonitoring(t).Status.Phase)
	})
}

func testDeletion(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "deletion")
	f.reconcileSuccessfully(t)

	// There is no finalizer, so cleanup relies entirely on the controller owner references
	// asserted in testOwnership. envtest runs no garbage collector, so the cascade itself can only
	// be verified on a real cluster (see the e2e follow-up).
	assert.Empty(t, f.storedPrometheusMonitoring(t).Finalizers)

	require.NoError(t, clt.Delete(t.Context(), f.pm))

	result, err := f.r.Reconcile(t.Context(), f.request())
	require.NoError(t, err, "reconciling a deleted PrometheusMonitoring must be a no-op")
	assert.Empty(t, result)
}

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

// newFixture creates a namespace, a token Secret, a Running DynaKube and a PrometheusMonitoring with
// explicit images for all three components, plus a Reconciler whose Dynatrace client factory is
// stubbed out. Explicit images keep the fleet-management image API out of the picture; resolving
// images from it is covered by the per-component unit tests.
func newFixture(t *testing.T, clt client.Client, name string) *fixture {
	t.Helper()

	ns := "pm-" + name
	integrationtests.CreateNamespace(t, clt, ns)

	integrationtests.CreateKubernetesObject(t, clt, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testDynaKubeName, Namespace: ns},
		Data: map[string][]byte{
			token.APIKey:        []byte(testAPIToken),
			token.PaaSKey:       []byte(testPaaSToken),
			token.DataIngestKey: []byte(testDataIngestToken),
		},
	})

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: testDynaKubeName, Namespace: ns},
		Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL},
		Status:     dynakube.DynaKubeStatus{Phase: status.Running},
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
		"dynaKubeRef":     testDynaKubeName,
		"gateway":         map[string]any{"image": testGatewayImage},
		"scraper":         map[string]any{"image": testScraperImage},
		"targetAllocator": map[string]any{"image": testTargetAllocatorImage},
	}, "spec"))

	integrationtests.CreateKubernetesObject(t, clt, obj)

	pm := &prometheusmonitoring.PrometheusMonitoring{}
	require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: obj.GetName(), Namespace: ns}, pm))

	return pm
}

// newStubbedReconciler is the production reconciler with only the Dynatrace API client factory
// replaced, so no test reaches out to a tenant. The image client always fails: every fixture pins
// explicit component images, so any fleet-management lookup means an image was not resolved from
// the spec, and a clean error is easier to read than a nil client.
func newStubbedReconciler(t *testing.T, clt client.Client) *Reconciler {
	t.Helper()

	imageClient := imagemock.NewClient(t)
	imageClient.EXPECT().GetComponentLatestInfo(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("fleet image API unavailable")).Maybe()

	r := NewReconciler(clt)
	r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, string, time.Duration) (*dynatrace.Client, error) {
		return &dynatrace.Client{Images: imageClient}, nil
	}

	return r
}

func (f *fixture) request() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: f.pm.Name, Namespace: f.ns}}
}

func (f *fixture) reconcile(t *testing.T) error {
	t.Helper()

	_, err := f.r.Reconcile(t.Context(), f.request())

	return err
}

func (f *fixture) reconcileSuccessfully(t *testing.T) {
	t.Helper()

	require.NoError(t, f.reconcile(t))
}

// updatePrometheusMonitoring persists spec changes made on the in-memory object. The reconciler reads the
// PrometheusMonitoring from the API server, so a spec change that is not written has no effect.
func (f *fixture) updatePrometheusMonitoring(t *testing.T) {
	t.Helper()

	stored := f.storedPrometheusMonitoring(t)
	stored.Spec = f.pm.Spec
	require.NoError(t, f.clt.Update(t.Context(), stored))

	f.pm.ResourceVersion = stored.ResourceVersion
}

func (f *fixture) storedPrometheusMonitoring(t *testing.T) *prometheusmonitoring.PrometheusMonitoring {
	t.Helper()

	stored := &prometheusmonitoring.PrometheusMonitoring{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKeyFromObject(f.pm), stored))

	return stored
}

func (f *fixture) tokenSecret(t *testing.T) *corev1.Secret {
	t.Helper()

	secret := &corev1.Secret{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKey{Name: f.dk.Tokens(), Namespace: f.ns}, secret))

	return secret
}

func (f *fixture) imageFor(componentName string) string {
	switch componentName {
	case "gateway":
		return testGatewayImage
	case "scraper":
		return testScraperImage
	default:
		return testTargetAllocatorImage
	}
}

func (f *fixture) configMap(t *testing.T, name string) *corev1.ConfigMap {
	t.Helper()

	cm := &corev1.ConfigMap{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, cm))

	return cm
}

func (f *fixture) service(t *testing.T, name string) *corev1.Service {
	t.Helper()

	svc := &corev1.Service{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, svc))

	return svc
}

func (f *fixture) deployment(t *testing.T, name string) *appsv1.Deployment {
	t.Helper()

	deploy := &appsv1.Deployment{}
	require.NoError(t, f.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: f.ns}, deploy))

	return deploy
}

func (f *fixture) gatewayStatefulSet(t *testing.T) *appsv1.StatefulSet {
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

// namespacedLists is one empty list per kind the controller could plausibly create. Listing more
// kinds than it actually creates is deliberate: a component that starts creating a
// PodDisruptionBudget, a NetworkPolicy or its own RBAC shows up as a diff instead of going
// unnoticed.
func namespacedLists() map[string]client.ObjectList {
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

// fixtureObjects is what newFixture itself puts into the namespace, in the kinds namespacedLists
// covers. The DynaKube and the PrometheusMonitoring are not among them.
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

	for kind, list := range namespacedLists() {
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
		tmpl, _ := c.podSpec(t, f)

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
