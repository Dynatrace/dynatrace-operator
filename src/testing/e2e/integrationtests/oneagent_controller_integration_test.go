//go:build integration
// +build integration

package integrationtests

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReconcileOneAgent_ReconcileOnEmptyEnvironment(t *testing.T) {
	oaName := "dynakube"

	e, err := newTestEnvironment()
	assert.NoError(t, err, "failed to start test environment")

	defer e.Stop()

	instance, err := e.AddOneAgent(oaName, &dynatracev1beta1.DynaKubeSpec{
		APIURL: DefaultTestAPIURL,
		Tokens: "token-test",
		OneAgent: dynatracev1beta1.OneAgentSpec{
			ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
		},
	})
	assert.NoError(t, err)

	_, err = e.Reconciler.Reconcile(context.TODO(), newReconciliationRequest(oaName))
	assert.NoError(t, err, "error reconciling")

	// Check if deamonset has been created and has correct namespace and name.
	dsActual := &appsv1.DaemonSet{}

	err = e.Client.Get(context.TODO(), types.NamespacedName{Name: instance.OneAgentDaemonsetName(), Namespace: DefaultTestNamespace}, dsActual)
	assert.NoError(t, err, "failed to get daemonset")

	assert.Equal(t, DefaultTestNamespace, dsActual.Namespace, "wrong namespace")
	assert.Equal(t, instance.OneAgentDaemonsetName(), dsActual.GetObjectMeta().GetName(), "wrong name")
	assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dsActual.Spec.Template.Spec.DNSPolicy, "DNS policy should ClusterFirst by default")
}
