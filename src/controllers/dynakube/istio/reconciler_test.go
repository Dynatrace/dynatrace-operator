package istio

import (
	"context"
	"os"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
)

const (
	DefaultTestNamespace = "dynatrace"

	testVirtualServiceName     = "dt-vs"
	testVirtualServiceHost     = "ENVIRONMENTID.live.dynatrace.com"
	testVirtualServiceProtocol = "https"
	testVirtualServicePort     = 443
)

func TestIstioClient_BuildDynatraceVirtualService(t *testing.T) {
	err := os.Setenv(kubeobjects.EnvPodNamespace, DefaultTestNamespace)
	if err != nil {
		t.Error("Failed to set environment variable")
	}

	commHosts := []dtclient.CommunicationHost{
		{Host: testVirtualServiceHost, Port: testVirtualServicePort, Protocol: testVirtualServiceProtocol},
	}

	vs := buildVirtualService(buildObjectMeta(testVirtualServiceName, DefaultTestNamespace), commHosts)
	ic := fakeistio.NewSimpleClientset(vs)
	vsList, err := ic.NetworkingV1alpha3().VirtualServices(DefaultTestNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("Failed to create VirtualService in %s namespace: %s", DefaultTestNamespace, err)
	}
	if len(vsList.Items) == 0 {
		t.Error("Expected items, got nil")
	}
	t.Logf("list of istio object %v", vsList.Items)
}

func TestReconcileIstio(t *testing.T) {
	t.Run(`reconciles istio objects correctly`, func(t *testing.T) {
		testReconcileIstio(t, true)
	})

	t.Run(`gracefully fail if istio is not installed`, func(t *testing.T) {
		testReconcileIstio(t, false)
	})
}

func testReconcileIstio(t *testing.T, enableIstioGVR bool) {
	serverUrl := "http://127.0.0.1:59842"
	port := 59842

	commHosts := []dtclient.CommunicationHost{{
		Host:     "localhost",
		Port:     uint32(port),
		Protocol: "http",
	},
	}
	virtualService := buildVirtualService(buildObjectMeta(testVirtualServiceName, DefaultTestNamespace), commHosts)
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dynakube",
			Namespace: DefaultTestNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: serverUrl,
		},
	}

	ist := fakeistio.NewSimpleClientset(virtualService)

	fakeDiscovery, ok := ist.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
	}

	if enableIstioGVR {
		fakeDiscovery.Resources = []*metav1.APIResourceList{{
			GroupVersion: IstioGVR,
		}}
	}

	reconciler := NewReconciler(
		&rest.Config{
			Host:    serverUrl,
			APIPath: "v1alpha3",
		},
		scheme.Scheme,
		ist,
	)
	updated, err := reconciler.Reconcile(instance, []dtclient.CommunicationHost{})

	assert.NoError(t, err)
	assert.Equal(t, enableIstioGVR, updated)

	update, err := reconciler.Reconcile(instance, []dtclient.CommunicationHost{})

	assert.NoError(t, err)
	assert.False(t, update)
}
