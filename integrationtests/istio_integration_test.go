// +build integration

package integrationtests

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconcileOneAgent_ReconcileIstio(t *testing.T) {
	e, err := newTestEnvironment()
	assert.NoError(t, err, "failed to start test environment")

	defer e.Stop()

	e.AddOneAgent("oneagent", &dynatracev1alpha1.OneAgentSpec{
		BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
			APIURL:      DefaultTestAPIURL,
			Tokens:      "token-test",
			EnableIstio: true,
		},
	})

	req := newReconciliationRequest("oneagent")

	// For the first reconciliation, we only create Istio objects for the API URL.
	_, err = e.Reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 1, 1)

	// Once the API URL is open, we create Istio objects for each communication endpoint.
	_, err = e.Reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 3, 3)

	// Add a new communication endpoint.
	e.CommunicationHosts = append(e.CommunicationHosts, "https://endpoint3.test.com/communication")
	_, err = e.Reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 4, 4)

	// Remove two communication endpoints.
	e.CommunicationHosts = e.CommunicationHosts[2:]
	_, err = e.Reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 2, 2)
}

func TestReconcileOneAgent_ReconcileIstioWithMultipleOneAgentObjects(t *testing.T) {
	e, err := newTestEnvironment()
	assert.NoError(t, err, "failed to start test environment")

	defer e.Stop()

	e.AddOneAgent("oneagent1", &dynatracev1alpha1.OneAgentSpec{
		BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
			APIURL:      DefaultTestAPIURL,
			Tokens:      "token-test",
			EnableIstio: true,
		},
	})

	e.AddOneAgent("oneagent2", &dynatracev1alpha1.OneAgentSpec{
		BaseOneAgentSpec: dynatracev1alpha1.BaseOneAgentSpec{
			APIURL:      DefaultTestAPIURL,
			Tokens:      "token-test",
			EnableIstio: true,
		},
	})

	req1 := newReconciliationRequest("oneagent1")
	req2 := newReconciliationRequest("oneagent2")

	// Operations on the CommunicationHosts list applies to both OneAgent objects, but that is fine, since that
	// is the most common use case as well, i.e., customers using multiple OneAgent objects for different
	// environments.

	// For the first reconciliation, we only create Istio objects for the API URL.
	_, err = e.Reconciler.Reconcile(context.TODO(), req1)
	assert.NoError(t, err, "failed to reconcile")
	_, err = e.Reconciler.Reconcile(context.TODO(), req2)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 2, 2)

	// Once the API URL is open, we create Istio objects for each communication endpoint.
	_, err = e.Reconciler.Reconcile(context.TODO(), req1)
	assert.NoError(t, err, "failed to reconcile")
	_, err = e.Reconciler.Reconcile(context.TODO(), req2)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 6, 6)

	// Add a new communication endpoint.
	e.CommunicationHosts = append(e.CommunicationHosts, "https://testendpoint.com/communication")
	_, err = e.Reconciler.Reconcile(context.TODO(), req1)
	assert.NoError(t, err, "failed to reconcile")
	_, err = e.Reconciler.Reconcile(context.TODO(), req2)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 8, 8)

	// Remove two communication endpoints.
	e.CommunicationHosts = e.CommunicationHosts[2:]
	_, err = e.Reconciler.Reconcile(context.TODO(), req1)
	assert.NoError(t, err, "failed to reconcile")
	_, err = e.Reconciler.Reconcile(context.TODO(), req2)
	assert.NoError(t, err, "failed to reconcile")
	assertIstioObjects(t, e.Client, 4, 4)
}

// assertIstioObjects confirms that we have the expected number of ServiceEntry and VirtualService objects, set by ese and evs respectively.
func assertIstioObjects(t *testing.T, c client.Client, ese, evs int) {
	var lst []string

	lst = findServiceEntries(t, c)
	assert.Equal(t, ese, len(lst), "unexpected number of ServiceEntry objects: %v", lst)

	lst = findVirtualServices(t, c)
	assert.Equal(t, evs, len(lst), "unexpected number of VirtualService objects: %v", lst)
}

func findServiceEntries(t *testing.T, c client.Client) []string {
	var lst istiov1alpha3.ServiceEntryList
	assert.NoError(t, c.List(context.TODO(), &lst), "failed to query ServiceEntry objects")

	var out []string
	for _, x := range lst.Items {
		out = append(out, x.Spec.Hosts...)
	}
	return out
}

func findVirtualServices(t *testing.T, c client.Client) []string {
	var lst istiov1alpha3.VirtualServiceList
	assert.NoError(t, c.List(context.TODO(), &lst), "failed to query VirtualService objects")

	var out []string
	for _, x := range lst.Items {
		out = append(out, x.Spec.Hosts...)
	}
	return out
}
