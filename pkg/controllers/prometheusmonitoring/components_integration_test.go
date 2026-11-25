// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/gateway"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/scraper"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/targetallocator"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// Integration tests for the top-level PrometheusMonitoring reconciler against a real API server.
//
// Scope is the orchestration this reconciler is responsible for: the preconditions it checks, the
// order it runs the component reconcilers in, that it stops at the first error, and the status it
// derives from what the components reported. Everything a component reconciler produces on its own
// (workload specs, configs, wiring, drift correction) is covered by that component's own tests, so
// the object assertions here are sanity checks that each component ran at all.
//
// envtest has no garbage collection, so ownership is asserted through owner references rather than
// through cascading deletion.

const (
	testAPIToken        = "dt0c01.INTEGRATIONAPITOKENVALUE"
	testDataIngestToken = "dt0c01.INTEGRATIONDATAINGESTTOKENVALUE"

	// The images the stubbed fleet management API hands out. Gateway and scraper are both plain
	// OTel Collectors and therefore share one image, exactly as they do in production.
	testCollectorImage       = "registry.example.com/dynatrace/otel-collector:1.2.3"
	testTargetAllocatorImage = "registry.example.com/dynatrace/target-allocator:1.2.3"

	testDynaKubeName = "dk"
	testAPIURL       = "https://tenant.dev.dynatracelabs.com/api"
	testClusterName  = "integration-cluster"
)

// TestReconcileOrchestration drives the full top-level Reconcile against a real API server.
// One envtest control plane is shared by all subtests; each subtest gets its own namespace so the
// object sets never overlap.
func TestReconcileOrchestration(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)

	t.Run("every component is deployed", func(t *testing.T) { testComponentsDeployed(t, clt) })
	t.Run("status", func(t *testing.T) { testStatus(t, clt) })
	t.Run("preconditions", func(t *testing.T) { testPreconditions(t, clt) })
	t.Run("dynakube phase is ignored", func(t *testing.T) { testDynaKubePhaseIsIgnored(t, clt) })
	t.Run("stops at the first failing component", func(t *testing.T) { testStopsAtFirstError(t, clt) })
	t.Run("multiple monitorings", func(t *testing.T) { testMultiplePrometheusMonitorings(t, clt) })
}

// All objects of a component share the component's name.
func gatewayObjects(pm *prometheusmonitoring.PrometheusMonitoring) []client.Object {
	name := pm.Gateway().GetStatefulSetName()

	return []client.Object{
		&corev1.ConfigMap{Name: name, Namespace: pm.Namespace},
		&appsv1.StatefulSet{Name: name, Namespace: pm.Namespace},
		&corev1.Service{Name: name, Namespace: pm.Namespace},
	}
}

func targetAllocatorObjects(pm *prometheusmonitoring.PrometheusMonitoring) []client.Object {
	name := pm.TargetAllocator().GetDeploymentName()

	return []client.Object{
		&corev1.ConfigMap{Name: name, Namespace: pm.Namespace},
		&appsv1.Deployment{Name: name, Namespace: pm.Namespace},
		&corev1.Service{Name: name, Namespace: pm.Namespace},
	}
}

// The scraper has no Service: nothing connects to it inbound.
func scraperObjects(pm *prometheusmonitoring.PrometheusMonitoring) []client.Object {
	name := pm.Scraper().GetDeploymentName()

	return []client.Object{
		&corev1.ConfigMap{Name: name, Namespace: pm.Namespace},
		&appsv1.Deployment{Name: name, Namespace: pm.Namespace},
	}
}

func allManagedObjects(pm *prometheusmonitoring.PrometheusMonitoring) []client.Object {
	return slices.Concat(gatewayObjects(pm), targetAllocatorObjects(pm), scraperObjects(pm))
}

// testComponentsDeployed is the sanity check that one successful reconcile ran all three component
// reconcilers: each one's main objects exist and belong to the PrometheusMonitoring. What those
// objects contain is asserted by the components' own tests.
func testComponentsDeployed(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "components")
	require.NoError(t, f.reconcile(t))

	for _, obj := range allManagedObjects(f.pm) {
		f.assertIsManaged(t, obj)
	}
}

// testStatus asserts the status the top-level reconciler assembles from what the components
// reported. The phase rules themselves are covered by Test_setPhase, so only the outcome of a real
// reconcile is tied to a phase here.
func testStatus(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "status")
	require.NoError(t, f.reconcile(t))

	stored := f.getStoredPrometheusMonitoring(t)

	t.Run("resolved images are recorded", func(t *testing.T) {
		assert.Equal(t, testCollectorImage, stored.Status.Gateway.ResolvedImage)
		assert.Equal(t, testCollectorImage, stored.Status.Scraper.ResolvedImage)
		assert.Equal(t, testTargetAllocatorImage, stored.Status.TargetAllocator.ResolvedImage)
	})

	t.Run("one availability condition per component plus the prerequisites", func(t *testing.T) {
		wantTypes := []string{
			prometheusmonitoring.PrerequisitesMet,
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
}

// testPreconditions covers the checks that make the reconcile bail out before it deploys anything.
// Which phase each one produces is already covered exhaustively and cheaply by TestReconcile and
// Test_setPhase against a fake client; what is only observable against a real apiserver is that the
// bail-out really happens before any object is written.
func testPreconditions(t *testing.T, clt client.Client) {
	preconditions := []struct {
		name      string
		namespace string
		prepare   func(t *testing.T, f *fixture)
		phase     status.DeploymentPhase
	}{
		{
			// A missing DynaKube is reported through the PrerequisitesMet condition and requeued.
			name:      "missing DynaKube",
			namespace: "err-no-dynakube",
			prepare:   func(t *testing.T, f *fixture) { require.NoError(t, clt.Delete(t.Context(), f.dk)) },
			phase:     status.Error,
		},
		{
			name:      "tenant URL missing",
			namespace: "err-no-api-url",
			prepare: func(t *testing.T, f *fixture) {
				f.dk.Spec.APIURL = ""
				require.NoError(t, clt.Update(t.Context(), f.dk))
			},
			phase: status.Error,
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
			f.assertObjectsAbsent(t, allManagedObjects(f.pm), "nothing may be deployed when the reconcile bails out")
		})
	}
}

// testDynaKubePhaseIsIgnored asserts that the components are deployed while the DynaKube is not Running,
// as long as it provides the tenant URL and the token secret.
func testDynaKubePhaseIsIgnored(t *testing.T, clt client.Client) {
	for _, phase := range []status.DeploymentPhase{status.Deploying, status.Error} {
		t.Run(string(phase), func(t *testing.T) {
			f := newFixture(t, clt, "dk-phase-"+strings.ToLower(string(phase)))
			f.dk.Status.Phase = phase
			require.NoError(t, f.dk.UpdateStatus(t.Context(), clt))

			require.NoError(t, f.reconcile(t))

			for _, obj := range allManagedObjects(f.pm) {
				f.assertIsManaged(t, obj)
			}
		})
	}
}

// testStopsAtFirstError pins the order the components are reconciled in and that the loop breaks on
// the first error. The gateway runs before the target allocator and the scraper runs after it, so a
// fleet management API that knows no target allocator image leaves a deployed gateway and no
// scraper behind. Nothing rolls back.
func testStopsAtFirstError(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "first-error")
	f.r = newReconcilerWithImages(t, clt, map[image.ComponentType]string{image.OTelCollector: testCollectorImage})

	require.ErrorContains(t, f.reconcile(t), "reconcile target allocator")

	// The gateway is reconciled before the target allocator, the scraper after it.
	for _, obj := range gatewayObjects(f.pm) {
		f.assertIsManaged(t, obj)
	}

	f.assertObjectsAbsent(t, scraperObjects(f.pm), "the scraper is reconciled after the target allocator")

	assert.Equal(t, status.Error, f.getStoredPrometheusMonitoring(t).Status.Phase)
}

func testMultiplePrometheusMonitorings(t *testing.T, clt client.Client) {
	const (
		dynaKubeName = "dk"
		image        = "registry.example.com/component:1.2.3"
	)

	dk := &dynakube.DynaKube{
		Name: dynaKubeName, Namespace: metav1.NamespaceDefault,
		Spec: dynakube.DynaKubeSpec{CustomPullSecret: "custom-pull-secret"},
	}

	newPM := func(name string) *prometheusmonitoring.PrometheusMonitoring {
		pm := &prometheusmonitoring.PrometheusMonitoring{
			Name: name, Namespace: metav1.NamespaceDefault,
			Spec: prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: dynaKubeName},
		}
		pm.Spec.Gateway.Image = image
		pm.Spec.Scraper.Image = image
		pm.Spec.TargetAllocator.Image = image
		pm.Spec.TargetAllocator.CustomResourceSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{"pm": name},
		}

		integrationtests.CreateKubernetesObject(t, clt, pm)

		return pm
	}

	pms := []*prometheusmonitoring.PrometheusMonitoring{newPM("pm-one"), newPM("pm-two")}

	for _, pm := range pms {
		require.NoError(t, (&gateway.Reconciler{Client: clt}).Reconcile(t.Context(), pm, dk, nil))
		require.NoError(t, (&targetallocator.Reconciler{Client: clt}).Reconcile(t.Context(), pm, dk, nil))
		require.NoError(t, (&scraper.Reconciler{Client: clt}).Reconcile(t.Context(), pm, dk, nil))
	}

	assertDistinctResourceNames(t, pms)
	assertServicesSelectOwnPods(t, clt, pms)
	assertAllocatorSelectors(t, clt, pms)
}

func assertDistinctResourceNames(t *testing.T, pms []*prometheusmonitoring.PrometheusMonitoring) {
	t.Helper()

	seen := map[string]string{}

	for _, pm := range pms {
		for kind, name := range map[string]string{
			"gateway":         pm.Gateway().GetStatefulSetName(),
			"scraper":         pm.Scraper().GetDeploymentName(),
			"targetallocator": pm.TargetAllocator().GetDeploymentName(),
		} {
			key := kind + "/" + name
			assert.NotContains(t, seen, key)
			seen[key] = pm.Name
		}
	}
}

// collectPodLabels returns the pod template labels of every workload, grouped by owning PrometheusMonitoring.
func collectPodLabels(t *testing.T, clt client.Client, pms []*prometheusmonitoring.PrometheusMonitoring) map[string][]map[string]string {
	t.Helper()

	podLabels := map[string][]map[string]string{}

	for _, pm := range pms {
		sts := &appsv1.StatefulSet{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.Gateway().GetStatefulSetName(), Namespace: metav1.NamespaceDefault}, sts))

		scraperDeploy := &appsv1.Deployment{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.Scraper().GetDeploymentName(), Namespace: metav1.NamespaceDefault}, scraperDeploy))

		allocatorDeploy := &appsv1.Deployment{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.TargetAllocator().GetDeploymentName(), Namespace: metav1.NamespaceDefault}, allocatorDeploy))

		podLabels[pm.Name] = []map[string]string{sts.Spec.Template.Labels, scraperDeploy.Spec.Template.Labels, allocatorDeploy.Spec.Template.Labels}
	}

	return podLabels
}

func assertServicesSelectOwnPods(t *testing.T, clt client.Client, pms []*prometheusmonitoring.PrometheusMonitoring) {
	t.Helper()

	podLabels := collectPodLabels(t, clt, pms)

	services := map[string]*prometheusmonitoring.PrometheusMonitoring{}

	for _, pm := range pms {
		services[pm.Gateway().GetStatefulSetName()] = pm
		services[pm.TargetAllocator().GetDeploymentName()] = pm
	}

	for name, owner := range services {
		svc := &corev1.Service{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: metav1.NamespaceDefault}, svc))
		require.NotEmpty(t, svc.Spec.Selector)

		selector := labels.SelectorFromSet(svc.Spec.Selector)

		matched := 0

		for pmName, pods := range podLabels {
			for _, l := range pods {
				if !selector.Matches(labels.Set(l)) {
					continue
				}

				assert.Equal(t, owner.Name, pmName, "service %s selects a pod of another PrometheusMonitoring", name)

				matched++
			}
		}

		assert.Equal(t, 1, matched, "service %s must select exactly one workload", name)
	}
}

func assertAllocatorSelectors(t *testing.T, clt client.Client, pms []*prometheusmonitoring.PrometheusMonitoring) {
	t.Helper()

	for _, pm := range pms {
		cm := &corev1.ConfigMap{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.TargetAllocator().GetDeploymentName(), Namespace: metav1.NamespaceDefault}, cm))

		var cfg targetallocator.Config
		require.NoError(t, yaml.Unmarshal([]byte(cm.Data["targetallocator.yaml"]), &cfg))

		assert.Equal(t, pm.Spec.TargetAllocator.CustomResourceSelector, cfg.PrometheusCR.PodMonitorSelector)
		assert.Equal(t, map[string]string{"pm": pm.Name}, cfg.PrometheusCR.PodMonitorSelector.MatchLabels)

		// The collector selector must match the pods of the own scraper only.
		scraperDeploy := &appsv1.Deployment{}
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: pm.Scraper().GetDeploymentName(), Namespace: metav1.NamespaceDefault}, scraperDeploy))

		collectorSelector, err := metav1.LabelSelectorAsSelector(cfg.CollectorSelector)
		require.NoError(t, err)
		assert.True(t, collectorSelector.Matches(labels.Set(scraperDeploy.Spec.Template.Labels)))

		for _, other := range pms {
			if other.Name == pm.Name {
				continue
			}

			otherScraper := &appsv1.Deployment{}
			require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: other.Scraper().GetDeploymentName(), Namespace: metav1.NamespaceDefault}, otherScraper))
			assert.False(t, collectorSelector.Matches(labels.Set(otherScraper.Spec.Template.Labels)),
				"allocator of %s selects the scraper of %s", pm.Name, other.Name)
		}
	}
}

// fixture is one fully provisioned PrometheusMonitoring scenario in its own namespace.
type fixture struct {
	clt client.Client
	r   *Reconciler
	pm  *prometheusmonitoring.PrometheusMonitoring
	dk  *dynakube.DynaKube
	ns  string
}

// newFixture creates a namespace, a token Secret, a DynaKube and a default
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
		Status:    dynakube.DynaKubeStatus{KubernetesClusterName: testClusterName},
	}
	integrationtests.CreateDynakube(t, clt, dk)

	// The API server applies the CRD defaults on create and Create writes the response back, so pm
	// is the defaulted object.
	pm := &prometheusmonitoring.PrometheusMonitoring{
		Name:      "pm",
		Namespace: ns,
		Spec:      prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeName: testDynaKubeName},
	}
	integrationtests.CreateKubernetesObject(t, clt, pm)

	return &fixture{clt: clt, r: newStubbedReconciler(t, clt), pm: pm, dk: dk, ns: ns}
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
	r.newDynatraceClient = func(context.Context, client.Reader, *dynakube.DynaKube, string, string, time.Duration) (*dynatrace.Client, error) {
		return &dynatrace.Client{Images: imageClient}, nil
	}

	return r
}

func (f *fixture) reconcile(t *testing.T) error {
	t.Helper()

	_, err := f.r.Reconcile(t.Context(), ctrl.Request{Name: f.pm.Name, Namespace: f.ns})

	return err
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

// assertIsManaged asserts that obj exists and is controlled by the PrometheusMonitoring. obj only
// needs to carry name and namespace; it holds the stored object afterwards.
func (f *fixture) assertIsManaged(t *testing.T, obj client.Object) {
	t.Helper()

	key := client.ObjectKeyFromObject(obj)
	require.NoErrorf(t, f.clt.Get(t.Context(), key, obj), "%T %s does not exist", obj, key)
	assert.Truef(t, metav1.IsControlledBy(obj, f.pm), "%T %s is not controlled by the PrometheusMonitoring", obj, key)
}

func (f *fixture) assertObjectsAbsent(t *testing.T, objects []client.Object, msg string) {
	t.Helper()

	for _, obj := range objects {
		key := client.ObjectKeyFromObject(obj)
		err := f.clt.Get(t.Context(), key, obj)
		assert.Truef(t, k8serrors.IsNotFound(err), "%T %s must not exist: %s", obj, key, msg)
	}
}
