// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the top-level PrometheusMonitoring reconciler against a real API server.
//
// Scope is the orchestration this reconciler is responsible for: the preconditions it checks, the
// order it runs the component reconcilers in, that it stops at the first error, and the status it
// derives from what the components reported. Everything a component reconciler produces on its own
// (workload specs, configs, wiring, drift correction) is covered by that component's own tests, so
// the object assertions here are sanity checks that each component ran at all.
//
// envtest has no kube-controller-manager and no kubelet, so there are no pods and no garbage
// collection. Ownership is asserted through owner references rather than cascading deletion.

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
	t.Run("stops at the first failing component", func(t *testing.T) { testStopsAtFirstError(t, clt) })
}

// managedObject identifies one object a component reconciler is expected to create.
type managedObject struct {
	kind string
	name string
}

func (o managedObject) String() string {
	return o.kind + "/" + o.name
}

// All objects of a component share the component's name.
func gatewayObjects(pm *prometheusmonitoring.PrometheusMonitoring) []managedObject {
	name := pm.Gateway().GetStatefulSetName()

	return []managedObject{{"ConfigMap", name}, {"StatefulSet", name}, {"Service", name}}
}

func targetAllocatorObjects(pm *prometheusmonitoring.PrometheusMonitoring) []managedObject {
	name := pm.TargetAllocator().GetDeploymentName()

	return []managedObject{{"ConfigMap", name}, {"Deployment", name}, {"Service", name}}
}

// The scraper has no Service: nothing connects to it inbound.
func scraperObjects(pm *prometheusmonitoring.PrometheusMonitoring) []managedObject {
	name := pm.Scraper().GetDeploymentName()

	return []managedObject{{"ConfigMap", name}, {"Deployment", name}}
}

func allManagedObjects(pm *prometheusmonitoring.PrometheusMonitoring) []managedObject {
	objects := gatewayObjects(pm)
	objects = append(objects, targetAllocatorObjects(pm)...)

	return append(objects, scraperObjects(pm)...)
}

// testComponentsDeployed is the sanity check that one successful reconcile ran all three component
// reconcilers: each one's main objects exist and belong to the PrometheusMonitoring. What those
// objects contain is asserted by the components' own tests.
func testComponentsDeployed(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "components")
	require.NoError(t, f.reconcile(t))

	for _, want := range allManagedObjects(f.pm) {
		obj := f.getManagedObject(t, want)
		assert.Truef(t, metav1.IsControlledBy(obj, f.pm), "%s is not controlled by the PrometheusMonitoring", want)
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
			f.assertObjectsAbsent(t, allManagedObjects(f.pm), "nothing may be deployed when the reconcile bails out")
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

	f.assertObjectsExist(t, gatewayObjects(f.pm), "the gateway is reconciled before the target allocator")
	f.assertObjectsAbsent(t, scraperObjects(f.pm), "the scraper is reconciled after the target allocator")

	assert.Equal(t, status.Error, f.getStoredPrometheusMonitoring(t).Status.Phase)
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

func (f *fixture) getManagedObject(t *testing.T, want managedObject) client.Object {
	t.Helper()

	obj := newObjectForKind(t, want.kind)
	require.NoErrorf(t, f.clt.Get(t.Context(), client.ObjectKey{Name: want.name, Namespace: f.ns}, obj), "%s does not exist", want)

	return obj
}

func (f *fixture) assertObjectsExist(t *testing.T, want []managedObject, msg string) {
	t.Helper()

	for _, object := range want {
		assert.NoErrorf(t, f.getObject(t, object), "%s must exist: %s", object, msg)
	}
}

func (f *fixture) assertObjectsAbsent(t *testing.T, want []managedObject, msg string) {
	t.Helper()

	for _, object := range want {
		assert.Truef(t, k8serrors.IsNotFound(f.getObject(t, object)), "%s must not exist: %s", object, msg)
	}
}

func (f *fixture) getObject(t *testing.T, want managedObject) error {
	t.Helper()

	return f.clt.Get(t.Context(), client.ObjectKey{Name: want.name, Namespace: f.ns}, newObjectForKind(t, want.kind))
}

func newObjectForKind(t *testing.T, kind string) client.Object {
	t.Helper()

	switch kind {
	case "ConfigMap":
		return &corev1.ConfigMap{}
	case "Service":
		return &corev1.Service{}
	case "Deployment":
		return &appsv1.Deployment{}
	case "StatefulSet":
		return &appsv1.StatefulSet{}
	default:
		require.Failf(t, "unhandled kind", "no getter for %s", kind)

		return nil
	}
}
